package handler_engine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"roboserver/auth"
	"roboserver/shared"
	"strings"
	"time"
)

// ReverseConnectRequest is the data payload for a connect_robot JSON-RPC request.
type ReverseConnectRequest struct {
	Port     int    `json:"port"`               // TCP port on the robot to connect to
	Protocol string `json:"protocol,omitempty"`  // "tcp" (default) or "udp"
	IP       string `json:"ip,omitempty"`        // Override IP (optional; defaults to last known)
}

// handleConnectRobotRequest processes a handler's request to connect to its robot.
func (hp *HandlerProcess) handleConnectRobotRequest(ctx context.Context, env *JSONRPCEnvelope) {
	// Parse the request
	reqData, err := json.Marshal(env.Data)
	if err != nil {
		hp.sendResponse(env.ID, nil, "invalid connect request data")
		return
	}
	var req ReverseConnectRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		hp.sendResponse(env.ID, nil, "invalid connect request: "+err.Error())
		return
	}

	if req.Port <= 0 {
		hp.sendResponse(env.ID, nil, "port is required")
		return
	}

	protocol := req.Protocol
	if protocol == "" {
		protocol = "tcp"
	}

	// Resolve robot IP
	ip := req.IP
	if ip == "" {
		ip = hp.resolveRobotIP(ctx)
	}
	if ip == "" {
		hp.sendResponse(env.ID, nil, "cannot determine robot IP")
		return
	}

	addr := fmt.Sprintf("%s:%d", ip, req.Port)

	switch protocol {
	case "tcp":
		hp.wg.Add(1)
		go func() {
			defer hp.wg.Done()
			hp.reverseConnectTCP(ctx, env.ID, addr)
		}()
	case "udp":
		hp.wg.Add(1)
		go func() {
			defer hp.wg.Done()
			hp.reverseConnectUDP(ctx, env.ID, addr)
		}()
	default:
		hp.sendResponse(env.ID, nil, "unsupported protocol: "+protocol)
	}
}

// resolveRobotIP looks up the robot's IP from heartbeat state or active session.
func (hp *HandlerProcess) resolveRobotIP(ctx context.Context) string {
	if hp.rds == nil {
		return hp.IP // Fall back to last known IP from spawn time
	}

	// Check heartbeat state first (most up-to-date)
	if hb, _ := hp.rds.GetHeartbeat(ctx, hp.UUID); hb != nil {
		return hb.IP
	}

	// Check active session
	if active, _ := hp.rds.GetActiveRobot(ctx, hp.UUID); active != nil {
		return active.IP
	}

	return hp.IP
}

// reverseConnectTCP dials the robot, performs a mutual AUTH handshake, then
// bridges the connection to the handler's stdin/stdout.
func (hp *HandlerProcess) reverseConnectTCP(ctx context.Context, requestID, addr string) {
	shared.DebugPrint("Reverse TCP connect to robot %s at %s", hp.UUID, addr)

	conn, err := net.DialTimeout("tcp", addr, shared.AppConfig.Timeouts.ReverseConnectTimeout())
	if err != nil {
		shared.DebugPrint("Reverse connect failed for %s: %v", hp.UUID, err)
		hp.sendResponse(requestID, nil, "connection failed: "+err.Error())
		return
	}
	defer conn.Close()

	// Perform the AUTH handshake as the server side
	// Send our identity so the robot knows who's connecting
	conn.Write([]byte(fmt.Sprintf("ROBOSERVER_CONNECT %s\n", hp.UUID)))

	// Wait for robot acknowledgment
	conn.SetReadDeadline(time.Now().Add(shared.AppConfig.Timeouts.HandshakeTimeout()))
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), 64*1024)

	if !scanner.Scan() {
		hp.sendResponse(requestID, nil, "robot did not respond")
		return
	}

	response := strings.TrimSpace(scanner.Text())
	if response != "CONNECT_OK" {
		hp.sendResponse(requestID, nil, "robot rejected connection: "+response)
		return
	}

	// Verify the robot's identity with a challenge-response
	nonce, err := auth.GenerateNonce()
	if err != nil {
		hp.sendResponse(requestID, nil, "failed to generate nonce")
		return
	}
	conn.Write([]byte(fmt.Sprintf("NONCE %s\n", nonce)))

	conn.SetReadDeadline(time.Now().Add(shared.AppConfig.Timeouts.HandshakeTimeout()))
	if !scanner.Scan() {
		hp.sendResponse(requestID, nil, "robot did not sign nonce")
		return
	}
	signature := strings.TrimSpace(scanner.Text())

	// Verify against stored public key
	robot, err := hp.db.GetRobotByUUID(ctx, hp.UUID)
	if err != nil {
		hp.sendResponse(requestID, nil, "robot not found in database")
		return
	}
	if err := auth.VerifySignature(robot.PublicKey, nonce, signature); err != nil {
		if err2 := auth.VerifyEd25519Hex(robot.PublicKey, nonce, signature); err2 != nil {
			hp.sendResponse(requestID, nil, "robot signature verification failed")
			return
		}
	}

	conn.SetReadDeadline(time.Time{}) // Clear deadline
	conn.Write([]byte("AUTH_OK\n"))

	// Update RobotSend to use the new connection (reject if one already exists)
	hp.mu.Lock()
	if hp.RobotSend != nil {
		hp.mu.Unlock()
		hp.sendResponse(requestID, nil, "robot already connected")
		return
	}
	hp.RobotSend = func(data []byte) error {
		data = append(data, '\n')
		_, err := conn.Write(data)
		return err
	}
	hp.mu.Unlock()

	// Connection established — notify handler
	hp.sendResponse(requestID, "connected", "")

	// Bridge: forward robot -> handler
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		hp.SendIncoming(scanner.Text())
	}

	// Connection closed
	hp.mu.Lock()
	hp.RobotSend = nil
	hp.mu.Unlock()

	hp.sendToScript(&DisconnectMessage{
		Type:   MsgTypeDisconnect,
		UUID:   hp.UUID,
		Reason: "reverse_tcp_closed",
	})
}

// reverseConnectUDP sets up a UDP "connection" to the robot and bridges it.
func (hp *HandlerProcess) reverseConnectUDP(ctx context.Context, requestID, addr string) {
	shared.DebugPrint("Reverse UDP connect to robot %s at %s", hp.UUID, addr)

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		hp.sendResponse(requestID, nil, "invalid UDP address: "+err.Error())
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		hp.sendResponse(requestID, nil, "UDP dial failed: "+err.Error())
		return
	}
	defer conn.Close()

	// Set up RobotSend for UDP (reject if one already exists)
	hp.mu.Lock()
	if hp.RobotSend != nil {
		hp.mu.Unlock()
		hp.sendResponse(requestID, nil, "robot already connected")
		return
	}
	hp.RobotSend = func(data []byte) error {
		_, err := conn.Write(data)
		return err
	}
	hp.mu.Unlock()

	hp.sendResponse(requestID, "connected", "")

	// Read loop for incoming UDP messages.
	// Use a short read deadline (1s) to periodically check context cancellation.
	// On timeout we loop back and re-check ctx.Done(); on real errors we break.
	const udpPollInterval = 1 * time.Second
	buf := make([]byte, 65535)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(udpPollInterval))
		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			break
		}
		hp.SendIncoming(string(buf[:n]))
	}

	hp.mu.Lock()
	hp.RobotSend = nil
	hp.mu.Unlock()
}
