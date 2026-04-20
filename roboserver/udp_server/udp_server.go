package udp_server

import (
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

// MaxUDPPacketSize is the maximum size of a single UDP datagram we'll read.
const MaxUDPPacketSize = 65535

// UDPPacket is the JSON envelope for all inbound UDP communication.
type UDPPacket struct {
	Type      string          `json:"type"`                // "auth", "heartbeat", "message"
	UUID      string          `json:"uuid"`                // Robot UUID
	Nonce     string          `json:"nonce,omitempty"`     // Echoed nonce for auth step 2
	Signature string          `json:"signature,omitempty"` // Cryptographic signature
	JWT       string          `json:"jwt,omitempty"`       // Session JWT (for messages)
	Payload   json.RawMessage `json:"payload,omitempty"`   // Message or heartbeat payload
}

// UDPResponse is the JSON envelope for all server responses.
type UDPResponse struct {
	Type   string `json:"type"`             // Response type (e.g. "auth_response")
	Status string `json:"status"`           // "ok", "nonce", "error"
	Nonce  string `json:"nonce,omitempty"`  // Nonce for auth challenge
	JWT    string `json:"jwt,omitempty"`    // Issued JWT
	Error  string `json:"error,omitempty"`  // Error message
}

type UDPServer_t struct {
	conn *net.UDPConn
	bus  comms.Bus
	db   database.DBManager
	ctx  context.Context
}

// Start initializes and runs the UDP server.
// Robots communicate via self-contained JSON packets:
//
//	{"type":"auth","uuid":"..."}                                          → auth step 1 (get nonce)
//	{"type":"auth","uuid":"...","nonce":"...","signature":"..."}          → auth step 2 (verify)
//	{"type":"heartbeat","uuid":"...","payload":"...","signature":"..."}   → signed heartbeat
//	{"type":"message","uuid":"...","jwt":"...","payload":"..."}           → message to handler
func Start(ctx context.Context, bus comms.Bus, db database.DBManager) error {
	port := shared.AppConfig.Server.UDPPort

	addr := &net.UDPAddr{
		Port: port,
		IP:   net.IPv4zero,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		shared.DebugPanic("Error starting UDP server: %v", err)
	}

	s := &UDPServer_t{
		conn: conn,
		bus:  bus,
		db:   db,
		ctx:  ctx,
	}

	go func() {
		shared.DebugPrint("UDP server listening on port %d", port)
		buf := make([]byte, MaxUDPPacketSize)
		for {
			// Use a short read deadline so we can check context cancellation
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					select {
					case <-ctx.Done():
						return
					default:
						continue
					}
				}
				select {
				case <-ctx.Done():
					return
				default:
					shared.DebugPrint("UDP read error: %v", err)
					continue
				}
			}

			// Copy packet data before handing off to goroutine
			packet := make([]byte, n)
			copy(packet, buf[:n])
			go s.handlePacket(packet, remoteAddr)
		}
	}()

	<-ctx.Done()
	shared.DebugPrint("Shutting down UDP server...")
	conn.Close()
	shared.DebugPrint("UDP server shut down gracefully")
	return nil
}

// handlePacket parses and routes a single UDP packet.
func (s *UDPServer_t) handlePacket(data []byte, addr *net.UDPAddr) {
	defer func() {
		if r := recover(); r != nil {
			shared.DebugPrint("UDP packet handler panic from %s: %v", addr, r)
		}
	}()
	var pkt UDPPacket
	if err := json.Unmarshal(data, &pkt); err != nil {
		s.sendResponse(addr, &UDPResponse{Type: "error", Status: "error", Error: "invalid JSON"})
		return
	}

	switch pkt.Type {
	case "auth":
		s.handleAuth(addr, &pkt)
	case "heartbeat":
		s.handleHeartbeat(addr, &pkt)
	case "message":
		s.handleMessage(addr, &pkt)
	default:
		s.sendResponse(addr, &UDPResponse{Type: "error", Status: "error", Error: "unknown packet type"})
	}
}

// handleAuth implements two-step challenge-response authentication over UDP.
// Step 1: Robot sends {"type":"auth","uuid":"..."} → server caches nonce, responds with nonce
// Step 2: Robot sends {"type":"auth","uuid":"...","nonce":"...","signature":"..."} → server verifies, responds with JWT
func (s *UDPServer_t) handleAuth(addr *net.UDPAddr, pkt *UDPPacket) {
	if s.db == nil {
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "no database"})
		return
	}

	pg := s.db.Postgres()
	rds := s.db.Redis()
	if pg == nil || rds == nil {
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "database unavailable"})
		return
	}

	uuid := pkt.UUID
	if uuid == "" {
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "uuid required"})
		return
	}

	// Step 1: No signature → look up robot, issue nonce
	if pkt.Signature == "" {
		robot, err := pg.GetRobotByUUID(s.ctx, uuid)
		if err != nil {
			s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "unknown robot"})
			return
		}
		if robot.IsBlacklisted {
			s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "blacklisted"})
			return
		}

		nonce, err := auth.GenerateNonce()
		if err != nil {
			s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "nonce generation failed"})
			return
		}

		// Cache nonce + robot info in Redis with short TTL (avoids double PG lookup)
		const nonceTTL = 30 * time.Second
		nonceKey := fmt.Sprintf("udp:nonce:%s", uuid)
		authCache := fmt.Sprintf("%s|%s|%s", nonce, robot.PublicKey, robot.DeviceType)
		rds.Client.Set(s.ctx, nonceKey, authCache, nonceTTL)

		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "nonce", Nonce: nonce})
		return
	}

	// Step 2: Signature provided → retrieve cached nonce+robot, verify, issue JWT
	nonceKey := fmt.Sprintf("udp:nonce:%s", uuid)
	authCache, err := rds.Client.GetDel(s.ctx, nonceKey).Result()
	if err != nil || authCache == "" {
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "no pending nonce (send auth request first)"})
		return
	}

	parts := strings.SplitN(authCache, "|", 3)
	if len(parts) != 3 {
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "corrupted auth state"})
		return
	}
	nonce, publicKey, deviceType := parts[0], parts[1], parts[2]

	if err := auth.VerifyRobotSignature(publicKey, nonce, pkt.Signature); err != nil {
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "signature verification failed"})
		return
	}

	ip := addr.IP.String()
	sessionID := auth.GenerateSessionID()
	jwt, err := auth.IssueSessionJWT(uuid, deviceType, ip, sessionID)
	if err != nil {
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "failed to issue JWT"})
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
	if err := rds.SetActiveRobot(s.ctx, activeRobot, ttl); err != nil {
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "failed to store session"})
		return
	}

	// Spawn or reattach handler
	robotSend := s.createRobotSend(uuid, addr)

	if existing, ok := handler_engine.HandlerManager.Get(uuid); ok {
		existing.Reattach(robotSend, ip, sessionID)
		shared.DebugPrint("UDP: Robot %s reattached to existing handler (PID %d)", uuid, existing.PID)
	} else if handler_engine.HandlerManager.TryStartSpawning(uuid) {
		var spawnErr error
		func() {
			defer handler_engine.HandlerManager.FinishSpawning(uuid)
			_, spawnErr = handler_engine.SpawnHandlerProcess(
				s.ctx,
				uuid, deviceType, ip, sessionID,
				pg, rds, s.bus,
				robotSend,
			)
		}()
		if spawnErr != nil {
			shared.DebugPrint("UDP: Failed to spawn handler for %s: %v", uuid, spawnErr)
			rds.RemoveActiveRobot(s.ctx, uuid)
			s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "handler spawn failed"})
			return
		}
	} else {
		// Another request is mid-spawn — wait briefly for it to appear.
		waitDeadline := time.Now().Add(shared.AppConfig.Timeouts.HandshakeTimeout())
		for time.Now().Before(waitDeadline) {
			if existing, ok := handler_engine.HandlerManager.Get(uuid); ok {
				existing.Reattach(robotSend, ip, sessionID)
				goto handlerReady
			}
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
		shared.DebugPrint("UDP: Handler for %s not available after wait", uuid)
		rds.RemoveActiveRobot(s.ctx, uuid)
		s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "error", Error: "handler unavailable"})
		return
	}
handlerReady:

	shared.DebugPrint("UDP: Robot %s authenticated from %s", uuid, shared.RedactIP(ip))
	s.sendResponse(addr, &UDPResponse{Type: "auth_response", Status: "ok", JWT: jwt})
}

// createRobotSend returns a callback that sends data to the robot via UDP.
func (s *UDPServer_t) createRobotSend(uuid string, addr *net.UDPAddr) func(data []byte) error {
	return func(data []byte) error {
		_, err := s.conn.WriteToUDP(data, addr)
		return err
	}
}

// handleHeartbeat processes a signed heartbeat from a UDP-connected robot.
func (s *UDPServer_t) handleHeartbeat(addr *net.UDPAddr, pkt *UDPPacket) {
	if s.db == nil {
		s.sendResponse(addr, &UDPResponse{Type: "heartbeat_response", Status: "error", Error: "no database"})
		return
	}

	pg := s.db.Postgres()
	rds := s.db.Redis()
	if pg == nil || rds == nil {
		s.sendResponse(addr, &UDPResponse{Type: "heartbeat_response", Status: "error", Error: "database unavailable"})
		return
	}

	if pkt.UUID == "" || pkt.Signature == "" || len(pkt.Payload) == 0 {
		s.sendResponse(addr, &UDPResponse{Type: "heartbeat_response", Status: "error", Error: "uuid, payload, and signature required"})
		return
	}

	ip := addr.IP.String()
	payloadJSON := string(pkt.Payload)

	result, err := auth.ProcessHeartbeat(s.ctx, pkt.UUID, payloadJSON, pkt.Signature, ip, pg, rds)
	if err != nil {
		shared.DebugPrint("UDP heartbeat failed for %s: %v", pkt.UUID, err)
		s.sendResponse(addr, &UDPResponse{Type: "heartbeat_response", Status: "error", Error: "heartbeat rejected"})
		return
	}

	if s.bus != nil {
		s.bus.PublishEvent(fmt.Sprintf("robot.%s.heartbeat", result.UUID), result)
	}

	s.sendResponse(addr, &UDPResponse{Type: "heartbeat_response", Status: "ok"})
}

// handleMessage forwards a JWT-authenticated message from a robot to its handler.
func (s *UDPServer_t) handleMessage(addr *net.UDPAddr, pkt *UDPPacket) {
	if pkt.UUID == "" || pkt.JWT == "" {
		s.sendResponse(addr, &UDPResponse{Type: "message_response", Status: "error", Error: "uuid and jwt required"})
		return
	}

	// Validate JWT and ensure it matches the claimed UUID
	claims, err := auth.ValidateSessionJWT(pkt.JWT)
	if err != nil || claims.Sub != pkt.UUID {
		s.sendResponse(addr, &UDPResponse{Type: "message_response", Status: "error", Error: "invalid or mismatched JWT"})
		return
	}

	hp, ok := handler_engine.HandlerManager.Get(pkt.UUID)
	if !ok {
		s.sendResponse(addr, &UDPResponse{Type: "message_response", Status: "error", Error: "no handler running"})
		return
	}

	hp.SendIncoming(string(pkt.Payload))
	s.sendResponse(addr, &UDPResponse{Type: "message_response", Status: "ok"})
}

// sendResponse marshals and sends a JSON response back to the given UDP address.
func (s *UDPServer_t) sendResponse(addr *net.UDPAddr, resp *UDPResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	s.conn.WriteToUDP(data, addr)
}
