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
	stderr io.ReadCloser
	cancel context.CancelFunc

	db  *database.PostgresHandler
	rds *database.RedisHandler
	bus comms.Bus

	mu     sync.Mutex
	closed bool

	// writeCh buffers messages for the dedicated stdin writer goroutine,
	// preventing mutex blocking when the handler script stalls (BUG-013).
	writeCh chan []byte

	// RobotSend is called to send data back to the robot's TCP connection.
	RobotSend func(data []byte) error

	// Bus subscription cancelers (cleaned up on Stop)
	subscriptions []func()

	// ForwardHeartbeats controls whether heartbeat events are forwarded to this handler.
	ForwardHeartbeats bool

	// wg tracks background goroutines (e.g., reverse connections) for clean shutdown.
	wg sync.WaitGroup
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

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
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
		stderr:     stderr,
		cancel:     cancel,
		db:         db,
		rds:        rds,
		bus:        bus,
		RobotSend:  robotSend,
		writeCh:    make(chan []byte, 256),
	}

	// Start dedicated stdin writer goroutine (decouples senders from blocking pipe writes)
	go hp.stdinWriter()

	// Store PID in Redis
	if rds != nil {
		active, _ := rds.GetActiveRobot(ctx, uuid)
		if active != nil {
			active.PID = hp.PID
			rds.SetActiveRobot(ctx, active, shared.AppConfig.Database.Redis.TTL())
		}
	}

	// Register in global handler map
	HandlerManager.Register(hp)

	shared.DebugPrint("Spawned handler process PID %d for robot %s (%s)", hp.PID, uuid, deviceType)

	// Send connect message to the handler script
	hp.sendToScript(&ConnectMessage{
		Type:       MsgTypeConnect,
		UUID:       uuid,
		DeviceType: deviceType,
		IP:         ip,
		SessionID:  sessionID,
	})

	// Subscribe to directed messages on the event bus (e.g., handler.{uuid}.message)
	hp.setupBusSubscriptions()

	// Start stdout listener (routes JSON-RPC envelopes)
	go hp.listenStdout(procCtx)

	// Start stderr listener (publishes handler log lines on the event bus)
	go hp.listenStderr(procCtx)

	return hp, nil
}

// setupBusSubscriptions sets up event bus subscriptions for this handler.
func (hp *HandlerProcess) setupBusSubscriptions() {
	if hp.bus == nil {
		return
	}

	// Subscribe to messages directed at this handler
	topic := fmt.Sprintf("handler.%s.message", hp.UUID)
	cancel, err := hp.bus.SubscribeEvent(topic, func(eventType string, data any) {
		hp.sendToScript(&EventMessage{
			Type:      MsgTypeEvent,
			EventType: eventType,
			Data:      data,
		})
	})
	if err == nil {
		hp.mu.Lock()
		hp.subscriptions = append(hp.subscriptions, cancel)
		hp.mu.Unlock()
	}
}

// Reattach reconnects a robot's TCP connection to this handler after a disconnect.
// Updates the RobotSend callback and sends a reconnect message to the handler script.
func (hp *HandlerProcess) Reattach(robotSend func(data []byte) error, ip, sessionID string) {
	hp.mu.Lock()
	hp.RobotSend = robotSend
	hp.IP = ip
	hp.SessionID = sessionID
	hp.mu.Unlock()

	hp.sendToScript(&ConnectMessage{
		Type:       MsgTypeConnect,
		UUID:       hp.UUID,
		DeviceType: hp.DeviceType,
		IP:         ip,
		SessionID:  sessionID,
	})
}

// SendIncoming forwards a message from the robot TCP connection to the handler's stdin.
func (hp *HandlerProcess) SendIncoming(payload string) {
	hp.sendToScript(&IncomingMessage{
		Type:    MsgTypeIncoming,
		UUID:    hp.UUID,
		Payload: payload,
	})
}

// SendDisconnect notifies the handler that the robot's TCP connection has closed,
// but does NOT kill the handler process. The handler may continue running for
// background tasks, reverse connections, etc.
func (hp *HandlerProcess) SendDisconnect(reason string) {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	if hp.closed {
		return
	}

	hp.RobotSend = nil // No longer connected

	msg := &DisconnectMessage{
		Type:   MsgTypeDisconnect,
		UUID:   hp.UUID,
		Reason: reason,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	data = append(data, '\n')

	select {
	case hp.writeCh <- data:
	default:
		shared.DebugPrint("Handler %s write buffer full, dropping disconnect message", hp.UUID)
	}
}

// Stop gracefully shuts down the handler process.
func (hp *HandlerProcess) Stop(reason string) {
	hp.mu.Lock()
	if hp.closed {
		hp.mu.Unlock()
		return
	}
	hp.closed = true

	// Send disconnect message while channel is still open (under lock to
	// prevent racing with close). This fixes a prior bug where the disconnect
	// was silently dropped because sendToScript checked closed=true.
	data, _ := json.Marshal(&DisconnectMessage{
		Type:   MsgTypeDisconnect,
		UUID:   hp.UUID,
		Reason: reason,
	})
	data = append(data, '\n')
	select {
	case hp.writeCh <- data:
	default:
	}
	hp.mu.Unlock()

	// Close the write channel — no more sends after closed=true,
	// so the writer goroutine will drain remaining messages and exit.
	close(hp.writeCh)

	// Give the script time to clean up
	done := make(chan struct{})
	go func() {
		hp.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		shared.DebugPrint("Handler process PID %d exited cleanly", hp.PID)
	case <-time.After(shared.AppConfig.Timeouts.ProcessKillTimeout()):
		shared.DebugPrint("Handler process PID %d did not exit in time, killing process group", hp.PID)
		// Kill the entire process group (negative PID) to prevent orphaned
		// child processes (e.g., python3, node) from leaking.
		syscall.Kill(-hp.cmd.Process.Pid, syscall.SIGKILL)
		hp.cancel()
	}

	// Close stdin — this also unblocks any pending write in the writer goroutine
	hp.stdin.Close()

	// Wait for background goroutines (reverse connections) to finish
	hp.wg.Wait()

	// Cancel all event bus subscriptions (copy under lock to avoid holding it during cancel calls)
	hp.mu.Lock()
	subs := make([]func(), len(hp.subscriptions))
	copy(subs, hp.subscriptions)
	hp.subscriptions = nil
	hp.mu.Unlock()
	for _, cancel := range subs {
		cancel()
	}

	// Unregister from global handler map
	HandlerManager.Unregister(hp.UUID)
}

func (hp *HandlerProcess) sendToScript(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		shared.DebugPrint("Failed to marshal message for handler %s: %v", hp.UUID, err)
		return
	}
	data = append(data, '\n')

	hp.mu.Lock()
	defer hp.mu.Unlock()
	if hp.closed {
		return
	}

	select {
	case hp.writeCh <- data:
	default:
		shared.DebugPrint("Handler %s write buffer full, dropping message", hp.UUID)
	}
}

// stdinWriter is a dedicated goroutine that drains the write channel and
// writes to the handler's stdin pipe. This decouples message senders from
// potentially blocking pipe writes, preventing mutex stalls (BUG-013).
func (hp *HandlerProcess) stdinWriter() {
	for data := range hp.writeCh {
		if _, err := hp.stdin.Write(data); err != nil {
			shared.DebugPrint("Failed to write to handler stdin %s: %v", hp.UUID, err)
			return
		}
	}
}

// listenStderr reads lines from the handler's stderr and publishes them as log events.
// Subscribers (e.g. WebSocket clients) can listen on "handler.{uuid}.log" for real-time logs.
func (hp *HandlerProcess) listenStderr(ctx context.Context) {
	scanner := bufio.NewScanner(hp.stderr)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		shared.DebugPrint("Handler %s stderr: %s", hp.UUID, line)

		if hp.bus != nil {
			hp.bus.PublishEvent(fmt.Sprintf("handler.%s.log", hp.UUID), map[string]string{
				"uuid":    hp.UUID,
				"line":    line,
				"stream":  "stderr",
			})
		}
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
			// Not valid JSON-RPC — treat as a log line from stdout
			logLine := string(line)
			shared.DebugPrint("Handler %s stdout: %s", hp.UUID, logLine)
			if hp.bus != nil {
				hp.bus.PublishEvent(fmt.Sprintf("handler.%s.log", hp.UUID), map[string]string{
					"uuid":   hp.UUID,
					"line":   logLine,
					"stream": "stdout",
				})
			}
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
	case TargetConfig:
		hp.handleConfigRequest(env)
	case TargetConnect:
		hp.handleConnectRobotRequest(ctx, env)
	default:
		shared.DebugPrint("Unknown target %q from handler %s", env.Target, hp.UUID)
		hp.sendResponse(env.ID, nil, "unknown target: "+env.Target)
	}
}

// redisDataKey generates the Redis key used to store arbitrary handler data.
func (hp *HandlerProcess) redisDataKey(key string) string {
	return fmt.Sprintf("handler:%s:data:%s", hp.UUID, key)
}

func (hp *HandlerProcess) handleDatabaseRequest(ctx context.Context, env *JSONRPCEnvelope) {
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

	case "list_robots":
		robots, err := hp.db.GetAllRobots(ctx)
		if err != nil {
			hp.sendResponse(env.ID, nil, err.Error())
			return
		}
		hp.sendResponse(env.ID, robots, "")

	case "get_robots_by_type":
		deviceType, ok := env.Data.(string)
		if !ok {
			hp.sendResponse(env.ID, nil, "data must be a device type string")
			return
		}
		robots, err := hp.db.GetRobotsByType(ctx, deviceType)
		if err != nil {
			hp.sendResponse(env.ID, nil, err.Error())
			return
		}
		hp.sendResponse(env.ID, robots, "")

	case "store_data":
		params, ok := env.Data.(map[string]interface{})
		if !ok {
			hp.sendResponse(env.ID, nil, "data must be an object with 'key' and 'value' fields")
			return
		}
		key, _ := params["key"].(string)
		if key == "" {
			hp.sendResponse(env.ID, nil, "key is required")
			return
		}
		value, err := json.Marshal(params["value"])
		if err != nil {
			hp.sendResponse(env.ID, nil, "failed to marshal value")
			return
		}
		// Store in Redis with handler-scoped key
		if hp.rds != nil {
			if err := hp.rds.Client.Set(ctx, hp.redisDataKey(key), value, 0).Err(); err != nil {
				hp.sendResponse(env.ID, nil, err.Error())
				return
			}
		}
		hp.sendResponse(env.ID, "stored", "")

	case "get_data":
		key, ok := env.Data.(string)
		if !ok {
			hp.sendResponse(env.ID, nil, "data must be a key string")
			return
		}
		if hp.rds != nil {
			val, err := hp.rds.Client.Get(ctx, hp.redisDataKey(key)).Result()
			if err != nil {
				hp.sendResponse(env.ID, nil, err.Error())
				return
			}
			// Return the raw JSON value
			var parsed interface{}
			if json.Unmarshal([]byte(val), &parsed) == nil {
				hp.sendResponse(env.ID, parsed, "")
			} else {
				hp.sendResponse(env.ID, val, "")
			}
		} else {
			hp.sendResponse(env.ID, nil, "redis not available")
		}

	case "delete_data":
		key, ok := env.Data.(string)
		if !ok {
			hp.sendResponse(env.ID, nil, "data must be a key string")
			return
		}
		if hp.rds != nil {
			if err := hp.rds.Client.Del(ctx, hp.redisDataKey(key)).Err(); err != nil {
				hp.sendResponse(env.ID, nil, err.Error())
				return
			}
		}
		hp.sendResponse(env.ID, "deleted", "")

	default:
		hp.sendResponse(env.ID, nil, "unknown database method: "+env.Method)
	}
}

func (hp *HandlerProcess) handleRobotRequest(env *JSONRPCEnvelope) {
	data, err := json.Marshal(env.Data)
	if err != nil {
		hp.sendResponse(env.ID, nil, "failed to marshal robot payload")
		return
	}

	if err := hp.SendToRobot(data); err != nil {
		hp.sendResponse(env.ID, nil, err.Error())
		return
	}
	hp.sendResponse(env.ID, "sent", "")
}

// SendToRobot safely copies the RobotSend callback under lock, then calls it.
// This prevents a data race with concurrent SendDisconnect/Reattach calls.
func (hp *HandlerProcess) SendToRobot(data []byte) error {
	hp.mu.Lock()
	send := hp.RobotSend
	hp.mu.Unlock()
	if send == nil {
		return fmt.Errorf("no robot connection available")
	}
	return send(data)
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

func (hp *HandlerProcess) handleConfigRequest(env *JSONRPCEnvelope) {
	switch env.Method {
	case "forward_heartbeats":
		enable, ok := env.Data.(bool)
		if !ok {
			hp.sendResponse(env.ID, nil, "data must be a boolean")
			return
		}
		if enable && !hp.ForwardHeartbeats {
			hp.enableHeartbeatForwarding()
		} else if !enable {
			hp.ForwardHeartbeats = false
		}
		hp.sendResponse(env.ID, hp.ForwardHeartbeats, "")

	case "subscribe":
		// Allow handlers to subscribe to arbitrary event bus topics
		topic, ok := env.Data.(string)
		if !ok {
			hp.sendResponse(env.ID, nil, "data must be an event type string")
			return
		}
		if hp.bus == nil {
			hp.sendResponse(env.ID, nil, "event bus not available")
			return
		}
		cancel, err := hp.bus.SubscribeEvent(topic, func(eventType string, data any) {
			hp.sendToScript(&EventMessage{
				Type:      MsgTypeEvent,
				EventType: eventType,
				Data:      data,
			})
		})
		if err != nil {
			hp.sendResponse(env.ID, nil, err.Error())
			return
		}
		hp.mu.Lock()
		hp.subscriptions = append(hp.subscriptions, cancel)
		hp.mu.Unlock()
		hp.sendResponse(env.ID, "subscribed", "")

	default:
		hp.sendResponse(env.ID, nil, "unknown config method: "+env.Method)
	}
}

// enableHeartbeatForwarding subscribes the handler to this robot's heartbeat events.
func (hp *HandlerProcess) enableHeartbeatForwarding() {
	if hp.bus == nil {
		return
	}

	topic := fmt.Sprintf("robot.%s.heartbeat", hp.UUID)
	cancel, err := hp.bus.SubscribeEvent(topic, func(eventType string, data any) {
		hp.sendToScript(&EventMessage{
			Type:      MsgTypeHeartbeat,
			EventType: eventType,
			Data:      data,
		})
	})
	if err == nil {
		hp.mu.Lock()
		hp.subscriptions = append(hp.subscriptions, cancel)
		hp.mu.Unlock()
		hp.ForwardHeartbeats = true
		shared.DebugPrint("Heartbeat forwarding enabled for handler %s", hp.UUID)
	}
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
