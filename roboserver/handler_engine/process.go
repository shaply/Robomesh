package handler_engine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/shared"
	"sync"
	"syscall"
	"time"
)

// HandlerProcess manages a single spawned handler script for one robot session.
type HandlerProcess struct {
	UUID       string
	DeviceType string
	IP         string
	SessionID  string
	PID        int

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cancel context.CancelFunc

	db  *database.PostgresHandler
	rds *database.RedisHandler
	bus comms.Bus

	mu     sync.Mutex
	closed bool

	// RobotSend is called to send data back to the robot's TCP connection.
	RobotSend func(data []byte) error
}

// SpawnHandlerProcess starts a handler script for an authenticated robot.
func SpawnHandlerProcess(
	ctx context.Context,
	uuid, deviceType, ip, sessionID string,
	db *database.PostgresHandler,
	rds *database.RedisHandler,
	bus comms.Bus,
	robotSend func(data []byte) error,
) (*HandlerProcess, error) {
	scriptPath, err := ResolveHandlerScript(deviceType)
	if err != nil {
		return nil, err
	}

	procCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(procCtx, "/bin/bash", scriptPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Put the handler in its own process group so we can kill the entire
	// tree (bash + any child python/node processes) on shutdown, preventing
	// orphaned processes from leaking memory and CPU.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set environment variables for the handler script
	cmd.Env = append(cmd.Environ(),
		fmt.Sprintf("ROBOT_UUID=%s", uuid),
		fmt.Sprintf("ROBOT_DEVICE_TYPE=%s", deviceType),
		fmt.Sprintf("ROBOT_IP=%s", ip),
		fmt.Sprintf("ROBOT_SESSION_ID=%s", sessionID),
	)

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start handler process: %w", err)
	}

	hp := &HandlerProcess{
		UUID:       uuid,
		DeviceType: deviceType,
		IP:         ip,
		SessionID:  sessionID,
		PID:        cmd.Process.Pid,
		cmd:        cmd,
		stdin:      stdin,
		stdout:     stdout,
		cancel:     cancel,
		db:         db,
		rds:        rds,
		bus:        bus,
		RobotSend:  robotSend,
	}

	// Store PID in Redis
	if rds != nil {
		active, _ := rds.GetActiveRobot(ctx, uuid)
		if active != nil {
			active.PID = hp.PID
			rds.SetActiveRobot(ctx, active, shared.AppConfig.Database.Redis.TTL())
		}
	}

	shared.DebugPrint("Spawned handler process PID %d for robot %s (%s)", hp.PID, uuid, deviceType)

	// Send connect message to the handler script
	hp.sendToScript(&ConnectMessage{
		Type:       MsgTypeConnect,
		UUID:       uuid,
		DeviceType: deviceType,
		IP:         ip,
		SessionID:  sessionID,
	})

	// Start stdout listener (routes JSON-RPC envelopes)
	go hp.listenStdout(procCtx)

	return hp, nil
}

// SendIncoming forwards a message from the robot TCP connection to the handler's stdin.
func (hp *HandlerProcess) SendIncoming(payload string) {
	hp.sendToScript(&IncomingMessage{
		Type:    MsgTypeIncoming,
		UUID:    hp.UUID,
		Payload: payload,
	})
}

// Stop gracefully shuts down the handler process.
func (hp *HandlerProcess) Stop(reason string) {
	hp.mu.Lock()
	if hp.closed {
		hp.mu.Unlock()
		return
	}
	hp.closed = true
	hp.mu.Unlock()

	// Send disconnect message
	hp.sendToScript(&DisconnectMessage{
		Type:   MsgTypeDisconnect,
		UUID:   hp.UUID,
		Reason: reason,
	})

	// Give the script time to clean up
	done := make(chan struct{})
	go func() {
		hp.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		shared.DebugPrint("Handler process PID %d exited cleanly", hp.PID)
	case <-time.After(10 * time.Second):
		shared.DebugPrint("Handler process PID %d did not exit in time, killing process group", hp.PID)
		// Kill the entire process group (negative PID) to prevent orphaned
		// child processes (e.g., python3, node) from leaking.
		syscall.Kill(-hp.cmd.Process.Pid, syscall.SIGKILL)
		hp.cancel()
	}

	hp.stdin.Close()

	// Clean up Redis
	if hp.rds != nil {
		hp.rds.RemoveActiveRobot(context.Background(), hp.UUID)
	}
}

func (hp *HandlerProcess) sendToScript(msg interface{}) {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	if hp.closed {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		shared.DebugPrint("Failed to marshal message for handler %s: %v", hp.UUID, err)
		return
	}
	data = append(data, '\n')
	if _, err := hp.stdin.Write(data); err != nil {
		shared.DebugPrint("Failed to write to handler stdin %s: %v", hp.UUID, err)
	}
}

// listenStdout reads JSON-RPC envelopes from the handler script's stdout and routes them.
func (hp *HandlerProcess) listenStdout(ctx context.Context) {
	scanner := bufio.NewScanner(hp.stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		var envelope JSONRPCEnvelope
		if err := json.Unmarshal(line, &envelope); err != nil {
			shared.DebugPrint("Invalid JSON from handler %s: %s", hp.UUID, string(line))
			continue
		}

		hp.routeEnvelope(ctx, &envelope)
	}

	if err := scanner.Err(); err != nil {
		shared.DebugPrint("Handler stdout error for %s: %v", hp.UUID, err)
	}
}

// routeEnvelope dispatches a JSON-RPC envelope to the appropriate target.
func (hp *HandlerProcess) routeEnvelope(ctx context.Context, env *JSONRPCEnvelope) {
	switch env.Target {
	case TargetDatabase:
		hp.handleDatabaseRequest(ctx, env)
	case TargetRobot:
		hp.handleRobotRequest(env)
	case TargetEventBus:
		hp.handleEventBusRequest(env)
	default:
		shared.DebugPrint("Unknown target %q from handler %s", env.Target, hp.UUID)
		hp.sendResponse(env.ID, nil, "unknown target: "+env.Target)
	}
}

func (hp *HandlerProcess) handleDatabaseRequest(ctx context.Context, env *JSONRPCEnvelope) {
	// For now, forward the raw query concept. Handlers can request data via
	// method names like "get_robot", "query", etc. This is extensible.
	switch env.Method {
	case "get_robot":
		uuid, ok := env.Data.(string)
		if !ok {
			hp.sendResponse(env.ID, nil, "data must be a robot UUID string")
			return
		}
		robot, err := hp.db.GetRobotByUUID(ctx, uuid)
		if err != nil {
			hp.sendResponse(env.ID, nil, err.Error())
			return
		}
		hp.sendResponse(env.ID, robot, "")
	default:
		hp.sendResponse(env.ID, nil, "unknown database method: "+env.Method)
	}
}

func (hp *HandlerProcess) handleRobotRequest(env *JSONRPCEnvelope) {
	if hp.RobotSend == nil {
		hp.sendResponse(env.ID, nil, "no robot connection available")
		return
	}

	data, err := json.Marshal(env.Data)
	if err != nil {
		hp.sendResponse(env.ID, nil, "failed to marshal robot payload")
		return
	}

	if err := hp.RobotSend(data); err != nil {
		hp.sendResponse(env.ID, nil, err.Error())
		return
	}
	hp.sendResponse(env.ID, "sent", "")
}

func (hp *HandlerProcess) handleEventBusRequest(env *JSONRPCEnvelope) {
	if hp.bus == nil {
		hp.sendResponse(env.ID, nil, "event bus not available")
		return
	}
	eventType := env.Method
	if eventType == "" {
		eventType = "handler_event"
	}
	hp.bus.PublishEvent(eventType, env.Data)
	hp.sendResponse(env.ID, "published", "")
}

func (hp *HandlerProcess) sendResponse(id string, data interface{}, errMsg string) {
	resp := JSONRPCEnvelope{
		ID:     id,
		Target: TargetResponse,
		Data:   data,
		Error:  errMsg,
	}
	hp.sendToScript(&resp)
}
