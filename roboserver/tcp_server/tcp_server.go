package tcp_server

import (
	"bufio"
	"context"
	"crypto/tls"
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

// MaxTCPMessageSize is the maximum allowed size for a single TCP message line.
// Prevents memory exhaustion from maliciously large payloads.
const MaxTCPMessageSize = 64 * 1024 // 64 KB

type TCPServer_t struct {
	bus          comms.Bus
	db           database.DBManager
	listener     net.Listener
	main_context context.Context
}

func Start(ctx context.Context, bus comms.Bus, dbManager database.DBManager) error {
	port := shared.AppConfig.Server.TCPPort

	var listener net.Listener
	var err error

	if shared.AppConfig.Server.TLS.Enabled {
		cert, tlsErr := tls.LoadX509KeyPair(
			shared.AppConfig.Server.TLS.CertFile,
			shared.AppConfig.Server.TLS.KeyFile,
		)
		if tlsErr != nil {
			shared.DebugPanic("Failed to load TLS certificate: %v", tlsErr)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		listener, err = tls.Listen("tcp", fmt.Sprintf(":%d", port), tlsConfig)
		shared.DebugPrint("TCP server using TLS")
	} else {
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	}
	if err != nil {
		shared.DebugPanic("Error starting TCP server: %v", err)
	}

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
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), MaxTCPMessageSize)

	for scanner.Scan() {
		message := strings.TrimSpace(scanner.Text())
		if message == "" {
			continue
		}
		shared.DebugPrint("Received: %s from %s", message, conn.RemoteAddr())

		switch {
		case message == "AUTH":
			s.handleAuthAndSession(conn, scanner)
			return
		case message == "REGISTER":
			s.handleRegisterAndSession(conn, scanner)
			return
		case strings.HasPrefix(message, "HEARTBEAT "):
			s.handleHeartbeat(conn, message)
			// Enter persistent heartbeat mode: keep reading subsequent heartbeats
			s.heartbeatLoop(conn, scanner)
			return
		default:
			conn.Write([]byte("ERROR EXPECTED_AUTH_OR_REGISTER\n"))
		}
	}

	if err := scanner.Err(); err != nil {
		shared.DebugPrint("Error reading from connection: %v", err)
	}
}

// readHandshakeInput sends a prompt to the robot, waits for a response with a timeout, and returns the trimmed response.
func (s *TCPServer_t) readHandshakeInput(conn net.Conn, scanner *bufio.Scanner, prompt string, emptyError string) (string, bool) {
	conn.Write([]byte(prompt + "\n"))
	conn.SetReadDeadline(time.Now().Add(shared.AppConfig.Timeouts.HandshakeTimeout()))
	if !scanner.Scan() {
		return "", false
	}
	val := strings.TrimSpace(scanner.Text())
	if val == "" {
		conn.Write([]byte("ERROR " + emptyError + "\n"))
		return "", false
	}
	return val, true
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
	pg := s.db.Postgres()
	if rds == nil {
		conn.Write([]byte("ERROR NO_DATABASE\n"))
		return
	}

	ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	// Step 1: Collect UUID
	uuid, ok := s.readHandshakeInput(conn, scanner, "REGISTER_CHALLENGE", "EMPTY_UUID")
	if !ok {
		return
	}

	// Check if UUID already exists in PostgreSQL (permanently registered)
	if pg != nil {
		if existing, _ := pg.GetRobotByUUID(s.main_context, uuid); existing != nil {
			conn.Write([]byte("ERROR UUID_ALREADY_REGISTERED\n"))
			return
		}
	}

	// Check if UUID already has an active session in Redis
	if active, _ := rds.GetActiveRobot(s.main_context, uuid); active != nil {
		conn.Write([]byte("ERROR UUID_ALREADY_ACTIVE\n"))
		return
	}

	// Check if UUID already has a pending registration
	if pending, _ := rds.GetPendingRobot(s.main_context, uuid); pending != nil {
		conn.Write([]byte("ERROR UUID_ALREADY_PENDING\n"))
		return
	}

	// Step 2: Collect device type
	deviceType, ok := s.readHandshakeInput(conn, scanner, "SEND_DEVICE_TYPE", "EMPTY_DEVICE_TYPE")
	if !ok {
		return
	}
	if !handler_engine.IsValidDeviceType(deviceType) {
		conn.Write([]byte("ERROR INVALID_DEVICE_TYPE\n"))
		return
	}

	// Step 3: Collect public key
	publicKey, ok := s.readHandshakeInput(conn, scanner, "SEND_PUBLIC_KEY", "EMPTY_PUBLIC_KEY")
	if !ok {
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
		conn.Write([]byte("ERROR REGISTRATION_FAILED\n"))
		return
	}

	// Step 5: Publish event for frontend/terminal notification
	eventData, err := json.Marshal(map[string]string{
		"device_id":  uuid,
		"ip":         ip,
		"robot_type": deviceType,
	})
	if err != nil {
		shared.DebugPrint("Failed to marshal registration event for %s: %v", uuid, err)
	} else if s.bus != nil {
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
	if err := rds.SetRobotPublicKey(s.main_context, uuid, publicKey, ttl); err != nil {
		shared.DebugPrint("Failed to store public key for %s: %v", uuid, err)
	}

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

// enterSessionMode either reattaches an existing handler or spawns a new one,
// then forwards all subsequent TCP lines to the handler.
// If isPersisted is false, the robot was registered via REGISTER and can send
// PERSIST to move to PostgreSQL.
func (s *TCPServer_t) enterSessionMode(conn net.Conn, scanner *bufio.Scanner, result *auth.HandshakeResult, isPersisted bool) {
	rds := s.db.Redis()
	pg := s.db.Postgres()

	// Create robotSend callback
	robotSend := func(data []byte) error {
		data = append(data, '\n')
		_, err := conn.Write(data)
		return err
	}

	// Try to reattach to an existing handler for this robot (e.g. after a TCP disconnect/reconnect)
	var hp *handler_engine.HandlerProcess
	if existing, ok := handler_engine.HandlerManager.Get(result.UUID); ok {
		existing.Reattach(robotSend, result.IP, result.SessionID)
		hp = existing
		shared.DebugPrint("Robot %s reconnected, reattached to existing handler (PID %d)", result.UUID, hp.PID)
	} else if handler_engine.HandlerManager.TryStartSpawning(result.UUID) {
		var err error
		hp, err = handler_engine.SpawnHandlerProcess(
			s.main_context,
			result.UUID, result.DeviceType, result.IP, result.SessionID,
			pg, rds, s.bus,
			robotSend,
		)
		handler_engine.HandlerManager.FinishSpawning(result.UUID)
		if err != nil {
			shared.DebugPrint("Failed to spawn handler for %s: %v", result.UUID, err)
			conn.Write([]byte("ERROR HANDLER_SPAWN_FAILED\n"))
			rds.RemoveActiveRobot(s.main_context, result.UUID)
			return
		}
		shared.DebugPrint("Handler spawned (PID %d) for robot %s, entering session mode", hp.PID, result.UUID)
	} else {
		// Another connection is currently spawning this handler — wait briefly then reattach
		shared.DebugPrint("Handler for %s is being spawned by another connection, waiting...", result.UUID)
		time.Sleep(2 * time.Second)
		if existing, ok := handler_engine.HandlerManager.Get(result.UUID); ok {
			existing.Reattach(robotSend, result.IP, result.SessionID)
			hp = existing
		} else {
			conn.Write([]byte("ERROR HANDLER_UNAVAILABLE\n"))
			return
		}
	}

	persisted := isPersisted

	// Session mode: forward all incoming TCP lines to the handler process,
	// but intercept PERSIST commands.
	for scanner.Scan() {
		select {
		case <-s.main_context.Done():
			hp.Stop("server_shutdown")
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

	// Connection closed — notify handler but don't kill it (Phase 3 keeps it alive)
	shared.DebugPrint("Robot %s TCP connection closed", result.UUID)
	hp.SendDisconnect("tcp_closed")
}

// handleHeartbeat processes a HEARTBEAT command.
// Format: HEARTBEAT <UUID> <signedPayloadJSON> <signatureHex>
func (s *TCPServer_t) handleHeartbeat(conn net.Conn, message string) {
	if s.db == nil {
		conn.Write([]byte("ERROR NO_DATABASE\n"))
		return
	}

	pg := s.db.Postgres()
	rds := s.db.Redis()
	if pg == nil || rds == nil {
		conn.Write([]byte("ERROR NO_DATABASE\n"))
		return
	}

	// Parse: HEARTBEAT <UUID> <payloadJSON> <signatureHex>
	// Use SplitN 3 to keep the payload+signature together, then split from the
	// right since signature (hex) never contains spaces but JSON payloads can.
	parts := strings.SplitN(message, " ", 3)
	if len(parts) != 3 {
		conn.Write([]byte("ERROR INVALID_HEARTBEAT_FORMAT\n"))
		return
	}

	uuid := parts[1]
	rest := parts[2] // "<payloadJSON> <signatureHex>"

	lastSpace := strings.LastIndex(rest, " ")
	if lastSpace == -1 {
		conn.Write([]byte("ERROR INVALID_HEARTBEAT_FORMAT\n"))
		return
	}

	payloadJSON := rest[:lastSpace]
	signature := rest[lastSpace+1:]
	ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	result, err := auth.ProcessHeartbeat(s.main_context, uuid, payloadJSON, signature, ip, pg, rds)
	if err != nil {
		shared.DebugPrint("Heartbeat failed for %s: %v", uuid, err)
		conn.Write([]byte("ERROR HEARTBEAT_REJECTED\n"))
		return
	}

	// Publish heartbeat event for any listeners (e.g., handlers with forward_heartbeats)
	if s.bus != nil {
		s.bus.PublishEvent(fmt.Sprintf("robot.%s.heartbeat", result.UUID), result)
	}

	conn.Write([]byte("HEARTBEAT_OK\n"))
}

// heartbeatLoop keeps reading heartbeat messages on a persistent connection.
func (s *TCPServer_t) heartbeatLoop(conn net.Conn, scanner *bufio.Scanner) {
	for scanner.Scan() {
		select {
		case <-s.main_context.Done():
			return
		default:
		}

		message := strings.TrimSpace(scanner.Text())
		if message == "" {
			continue
		}

		if strings.HasPrefix(message, "HEARTBEAT ") {
			s.handleHeartbeat(conn, message)
		} else {
			conn.Write([]byte("ERROR EXPECTED_HEARTBEAT\n"))
		}
	}
}

// handlePersist copies a robot's data from the active Redis session into
// PostgreSQL for permanent storage. Requires the robot's public key to be
// available (stored during REGISTER flow in the active session or retrieved).
func (s *TCPServer_t) handlePersist(conn net.Conn, result *auth.HandshakeResult, rds *database.RedisHandler, pg *database.PostgresHandler) {
	if pg == nil || rds == nil {
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
		conn.Write([]byte("ERROR PERSIST_FAILED\n"))
		return
	}

	shared.DebugPrint("Robot %s persisted to PostgreSQL", result.UUID)
	conn.Write([]byte("PERSIST_OK\n"))
}
