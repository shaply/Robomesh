package tcp_server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"roboserver/auth"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/handler_engine"
	"roboserver/shared"
	"strings"
	"time"
)

type TCPServer_t struct {
	bus          comms.Bus
	db           database.DBManager
	listener     net.Listener
	main_context context.Context
}

func Start(ctx context.Context, bus comms.Bus, dbManager database.DBManager) error {
	port := shared.AppConfig.Server.TCPPort

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		shared.DebugPanic("Error starting TCP server:", err)
	}
	defer listener.Close()

	s := &TCPServer_t{
		bus:          bus,
		db:           dbManager,
		listener:     listener,
		main_context: ctx,
	}

	go func() {
		shared.DebugPrint("TCP server listening on port %d", port)
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			shared.DebugPrint("Accepted connection from %s", conn.RemoteAddr())
			go s.handleConnection(conn)
		}
	}()

	<-ctx.Done()
	shared.DebugPrint("Shutting down TCP server...")
	if err := listener.Close(); err != nil {
		shared.DebugPrint("Error shutting down TCP server:", err)
		return fmt.Errorf("error shutting down TCP server: %w", err)
	}
	shared.DebugPrint("TCP server has shut down gracefully.")
	return nil
}

// handleConnection dispatches to AUTH or REGISTER based on the first command.
func (s *TCPServer_t) handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		message := strings.TrimSpace(scanner.Text())
		if message == "" {
			continue
		}
		shared.DebugPrint("Received: %s from %s", message, conn.RemoteAddr())

		switch message {
		case "AUTH":
			s.handleAuthAndSession(conn, scanner)
			return
		case "REGISTER":
			s.handleRegisterAndSession(conn, scanner)
			return
		default:
			conn.Write([]byte("ERROR EXPECTED_AUTH_OR_REGISTER\n"))
		}
	}

	if err := scanner.Err(); err != nil {
		shared.DebugPrint("Error reading from connection: %v", err)
	}
}

// handleAuthAndSession performs the cryptographic handshake against PostgreSQL,
// spawns a handler process, and enters session mode.
func (s *TCPServer_t) handleAuthAndSession(conn net.Conn, scanner *bufio.Scanner) {
	if s.db == nil {
		conn.Write([]byte("ERROR NO_DATABASE\n"))
		shared.DebugPrint("AUTH failed: database manager not initialized")
		return
	}

	pg := s.db.Postgres()
	rds := s.db.Redis()
	if pg == nil || rds == nil {
		conn.Write([]byte("ERROR NO_DATABASE\n"))
		return
	}

	// Perform cryptographic handshake (looks up robot in PostgreSQL)
	result, err := auth.PerformHandshake(s.main_context, conn, pg, rds)
	if err != nil {
		shared.DebugPrint("Handshake failed: %v", err)
		return
	}

	shared.DebugPrint("Robot %s (%s) authenticated, spawning handler", result.UUID, result.DeviceType)
	s.enterSessionMode(conn, scanner, result, true)
}

// handleRegisterAndSession collects robot info, waits for user approval via
// Redis pub/sub, then enters session mode if accepted. The robot is stored
// only in Redis (ephemeral) unless it later sends PERSIST.
//
// Protocol:
//   Server: REGISTER_CHALLENGE
//   Robot:  UUID
//   Server: SEND_DEVICE_TYPE
//   Robot:  <device_type>
//   Server: SEND_PUBLIC_KEY
//   Robot:  <public_key_hex>
//   Server: REGISTER_PENDING (waiting for user approval)
//   Server: REGISTER_OK <jwt>  |  REGISTER_REJECTED
func (s *TCPServer_t) handleRegisterAndSession(conn net.Conn, scanner *bufio.Scanner) {
	rds := s.db.Redis()
	if rds == nil {
		conn.Write([]byte("ERROR NO_DATABASE\n"))
		return
	}

	ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	// Step 1: Collect UUID
	conn.Write([]byte("REGISTER_CHALLENGE\n"))
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if !scanner.Scan() {
		return
	}
	uuid := strings.TrimSpace(scanner.Text())
	if uuid == "" {
		conn.Write([]byte("ERROR EMPTY_UUID\n"))
		return
	}

	// Step 2: Collect device type
	conn.Write([]byte("SEND_DEVICE_TYPE\n"))
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if !scanner.Scan() {
		return
	}
	deviceType := strings.TrimSpace(scanner.Text())
	if deviceType == "" {
		conn.Write([]byte("ERROR EMPTY_DEVICE_TYPE\n"))
		return
	}

	// Step 3: Collect public key
	conn.Write([]byte("SEND_PUBLIC_KEY\n"))
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if !scanner.Scan() {
		return
	}
	publicKey := strings.TrimSpace(scanner.Text())
	if publicKey == "" {
		conn.Write([]byte("ERROR EMPTY_PUBLIC_KEY\n"))
		return
	}

	// Clear read deadline for the wait phase
	conn.SetReadDeadline(time.Time{})

	// Step 4: Store as pending in Redis
	pending := &database.PendingRobot{
		UUID:        uuid,
		IP:          ip,
		DeviceType:  deviceType,
		PublicKey:   publicKey,
		RequestedAt: time.Now().Unix(),
	}

	pendingTTL := 5 * time.Minute // Pending registrations expire after 5 minutes
	if err := rds.SetPendingRobot(s.main_context, pending, pendingTTL); err != nil {
		shared.DebugPrint("Failed to store pending robot %s: %v", uuid, err)
		conn.Write([]byte(fmt.Sprintf("ERROR %v\n", err)))
		return
	}

	// Step 5: Publish event for frontend/terminal notification
	eventData, _ := json.Marshal(map[string]string{
		"device_id":  uuid,
		"ip":         ip,
		"robot_type": deviceType,
	})
	if s.bus != nil {
		s.bus.PublishEvent("robot.registering", string(eventData))
	}

	conn.Write([]byte("REGISTER_PENDING\n"))
	shared.DebugPrint("Robot %s pending registration approval", uuid)

	// Step 6: Wait for accept/reject via comms bus (Redis pub/sub in local mode)
	waitCtx, waitCancel := context.WithTimeout(s.main_context, pendingTTL)
	defer waitCancel()

	accepted, err := s.bus.WaitForRegistrationResponse(waitCtx, uuid)
	rds.RemovePendingRobot(s.main_context, uuid)

	if err != nil {
		shared.DebugPrint("Registration wait expired for %s: %v", uuid, err)
		conn.Write([]byte("ERROR REGISTRATION_TIMEOUT\n"))
		return
	}

	if !accepted {
		shared.DebugPrint("Robot %s registration rejected", uuid)
		conn.Write([]byte("REGISTER_REJECTED\n"))
		return
	}

	// Step 7: Accepted — issue JWT, store as active in Redis
	sessionID := auth.GenerateSessionID()
	jwt, err := auth.IssueSessionJWT(uuid, deviceType, ip, sessionID)
	if err != nil {
		conn.Write([]byte("ERROR SERVER_ERROR\n"))
		return
	}

	ttl := shared.AppConfig.Database.Redis.TTL()
	activeRobot := &database.ActiveRobot{
		UUID:        uuid,
		IP:          ip,
		DeviceType:  deviceType,
		SessionJWT:  jwt,
		ConnectedAt: time.Now().Unix(),
	}
	if err := rds.SetActiveRobot(s.main_context, activeRobot, ttl); err != nil {
		conn.Write([]byte("ERROR SERVER_ERROR\n"))
		return
	}

	// Store public key in Redis so PERSIST can copy it to PostgreSQL later
	rds.SetRobotPublicKey(s.main_context, uuid, publicKey, ttl)

	conn.Write([]byte(fmt.Sprintf("REGISTER_OK %s\n", jwt)))
	shared.DebugPrint("Robot %s registration accepted, entering session mode", uuid)

	result := &auth.HandshakeResult{
		UUID:       uuid,
		DeviceType: deviceType,
		IP:         ip,
		SessionJWT: jwt,
		SessionID:  sessionID,
	}
	s.enterSessionMode(conn, scanner, result, false)
}

// enterSessionMode starts the heartbeat, spawns a handler process, and
// forwards all subsequent TCP lines to it. If isPersisted is false, the
// robot was registered via REGISTER and can send PERSIST to move to PostgreSQL.
func (s *TCPServer_t) enterSessionMode(conn net.Conn, scanner *bufio.Scanner, result *auth.HandshakeResult, isPersisted bool) {
	rds := s.db.Redis()
	pg := s.db.Postgres()

	// Start heartbeat loop
	heartbeatDone := make(chan struct{})
	go auth.StartHeartbeatLoop(s.main_context, rds, result.UUID, heartbeatDone)

	// Create robotSend callback
	robotSend := func(data []byte) error {
		data = append(data, '\n')
		_, err := conn.Write(data)
		return err
	}

	// Spawn handler process
	hp, err := handler_engine.SpawnHandlerProcess(
		s.main_context,
		result.UUID, result.DeviceType, result.IP, result.SessionID,
		pg, rds, s.bus,
		robotSend,
	)
	if err != nil {
		shared.DebugPrint("Failed to spawn handler for %s: %v", result.UUID, err)
		conn.Write([]byte(fmt.Sprintf("ERROR HANDLER_FAILED %v\n", err)))
		close(heartbeatDone)
		rds.RemoveActiveRobot(s.main_context, result.UUID)
		return
	}

	shared.DebugPrint("Handler spawned (PID %d) for robot %s, entering session mode", hp.PID, result.UUID)

	persisted := isPersisted

	// Session mode: forward all incoming TCP lines to the handler process,
	// but intercept PERSIST commands.
	for scanner.Scan() {
		select {
		case <-s.main_context.Done():
			hp.Stop("server_shutdown")
			close(heartbeatDone)
			return
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Intercept PERSIST command
		if line == "PERSIST" && !persisted {
			s.handlePersist(conn, result, rds, pg)
			persisted = true
			continue
		}

		hp.SendIncoming(line)
	}

	// Connection closed — tear everything down
	shared.DebugPrint("Robot %s disconnected", result.UUID)
	hp.Stop("disconnected")
	close(heartbeatDone)
}

// handlePersist copies a robot's data from the active Redis session into
// PostgreSQL for permanent storage. Requires the robot's public key to be
// available (stored during REGISTER flow in the active session or retrieved).
func (s *TCPServer_t) handlePersist(conn net.Conn, result *auth.HandshakeResult, rds *database.RedisHandler, pg *database.PostgresHandler) {
	if pg == nil {
		conn.Write([]byte("ERROR NO_DATABASE\n"))
		return
	}

	// Check if already persisted in PostgreSQL
	existing, _ := pg.GetRobotByUUID(s.main_context, result.UUID)
	if existing != nil {
		conn.Write([]byte("PERSIST_OK ALREADY_PERSISTED\n"))
		return
	}

	// We need the public key — get it from the active robot's extended data in Redis
	publicKey, err := rds.GetRobotPublicKey(s.main_context, result.UUID)
	if err != nil || publicKey == "" {
		conn.Write([]byte("ERROR NO_PUBLIC_KEY\n"))
		shared.DebugPrint("PERSIST failed for %s: no public key found", result.UUID)
		return
	}

	// Store in PostgreSQL
	if err := pg.RegisterRobot(s.main_context, result.UUID, publicKey, result.DeviceType); err != nil {
		shared.DebugPrint("PERSIST failed for %s: %v", result.UUID, err)
		conn.Write([]byte(fmt.Sprintf("ERROR PERSIST_FAILED %v\n", err)))
		return
	}

	shared.DebugPrint("Robot %s persisted to PostgreSQL", result.UUID)
	conn.Write([]byte("PERSIST_OK\n"))
}
