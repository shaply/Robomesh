package auth

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"roboserver/database"
	"roboserver/shared"
	"strings"
	"time"
)

// HandshakeResult contains the outcome of a successful cryptographic handshake.
type HandshakeResult struct {
	UUID       string
	DeviceType string
	IP         string
	SessionJWT string
	SessionID  string
}

// PerformHandshake executes the full challenge-response authentication flow:
//  1. Robot sends UUID
//  2. Server looks up robot in PostgreSQL, checks blacklist
//  3. Server generates and sends a random Nonce
//  4. Robot signs the Nonce with its private key and returns the signature
//  5. Server verifies signature against stored public key
//  6. Server issues a session JWT and registers in Redis
func PerformHandshake(ctx context.Context, conn net.Conn, db *database.PostgresHandler, rds *database.RedisHandler) (*HandshakeResult, error) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), 64*1024)
	ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	// Step 1: Receive UUID
	conn.Write([]byte("AUTH_CHALLENGE\n"))
	conn.SetReadDeadline(time.Now().Add(shared.AppConfig.Timeouts.HandshakeTimeout()))
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read UUID: %w", scanner.Err())
	}
	uuid := strings.TrimSpace(scanner.Text())
	if uuid == "" {
		conn.Write([]byte("ERROR EMPTY_UUID\n"))
		return nil, fmt.Errorf("empty UUID received")
	}

	// Step 2: Look up robot in PostgreSQL
	robot, err := db.GetRobotByUUID(ctx, uuid)
	if err != nil {
		conn.Write([]byte("ERROR UNKNOWN_ROBOT\n"))
		return nil, fmt.Errorf("robot not found: %s", uuid)
	}
	if robot.IsBlacklisted {
		conn.Write([]byte("ERROR BLACKLISTED\n"))
		return nil, fmt.Errorf("robot is blacklisted: %s", uuid)
	}

	// Step 3: Generate and send Nonce
	nonce, err := GenerateNonce()
	if err != nil {
		conn.Write([]byte("ERROR SERVER_ERROR\n"))
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	conn.Write([]byte(fmt.Sprintf("NONCE %s\n", nonce)))

	// Step 4: Receive signature
	conn.SetReadDeadline(time.Now().Add(shared.AppConfig.Timeouts.HandshakeTimeout()))
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read signature: %w", scanner.Err())
	}
	signature := strings.TrimSpace(scanner.Text())
	if signature == "" {
		conn.Write([]byte("ERROR EMPTY_SIGNATURE\n"))
		return nil, fmt.Errorf("empty signature received")
	}

	// Step 5: Verify signature
	if err := VerifyRobotSignature(robot.PublicKey, nonce, signature); err != nil {
		conn.Write([]byte("ERROR INVALID_SIGNATURE\n"))
		return nil, fmt.Errorf("signature verification failed for %s: %w", uuid, err)
	}

	// Step 6: Issue JWT and register in Redis
	sessionID := GenerateSessionID()
	jwt, err := IssueSessionJWT(uuid, robot.DeviceType, ip, sessionID)
	if err != nil {
		conn.Write([]byte("ERROR SERVER_ERROR\n"))
		return nil, fmt.Errorf("failed to issue JWT: %w", err)
	}

	ttl := shared.AppConfig.Database.Redis.TTL()
	activeRobot := &database.ActiveRobot{
		UUID:        uuid,
		IP:          ip,
		DeviceType:  robot.DeviceType,
		SessionJWT:  jwt,
		ConnectedAt: time.Now().Unix(),
	}
	if err := rds.SetActiveRobot(ctx, activeRobot, ttl); err != nil {
		conn.Write([]byte("ERROR SERVER_ERROR\n"))
		return nil, fmt.Errorf("failed to register in Redis: %w", err)
	}

	conn.Write([]byte(fmt.Sprintf("AUTH_OK %s\n", jwt)))
	shared.DebugPrint("Robot %s authenticated successfully from %s", uuid, shared.RedactIP(ip))

	return &HandshakeResult{
		UUID:       uuid,
		DeviceType: robot.DeviceType,
		IP:         ip,
		SessionJWT: jwt,
		SessionID:  sessionID,
	}, nil
}

// VerifyRobotSignature tries PEM first, then raw hex Ed25519.
func VerifyRobotSignature(publicKey, nonce, signature string) error {
	// Try PEM-encoded key first
	err := VerifySignature(publicKey, nonce, signature)
	if err == nil {
		return nil
	}
	// Fall back to raw hex Ed25519
	return VerifyEd25519Hex(publicKey, nonce, signature)
}
