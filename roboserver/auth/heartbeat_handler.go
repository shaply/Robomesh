package auth

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"roboserver/database"
	"roboserver/shared"
	"time"
)

// HeartbeatPayload is the signed JSON payload sent by a robot.
type HeartbeatPayload struct {
	Seq       int64           `json:"seq"`                  // Sequence number (must increase)
	TTL       int             `json:"ttl,omitempty"`        // Optional custom TTL in seconds
	ExtraData json.RawMessage `json:"extra_data,omitempty"` // Optional additional data
}

// MaxHeartbeatTTL caps how long a robot can request its heartbeat state stay
// in Redis. Prevents a misbehaving (or hostile) robot from pinning state
// forever with an absurd TTL and filling Redis.
const MaxHeartbeatTTL = 6 * time.Hour

// HeartbeatResult contains the processed heartbeat information.
type HeartbeatResult struct {
	UUID    string
	IP      string
	Payload *HeartbeatPayload
}

// ProcessHeartbeat verifies a signed heartbeat and updates Redis state.
// The heartbeat format is: UUID + signed JSON payload.
// The signature is verified against the robot's public key from PostgreSQL.
func ProcessHeartbeat(ctx context.Context, uuid, payloadJSON, signature, ip string, pg *database.PostgresHandler, rds *database.RedisHandler) (*HeartbeatResult, error) {
	// Look up the robot's public key
	robot, err := pg.GetRobotByUUID(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("unknown robot: %s", uuid)
	}
	if robot.IsBlacklisted {
		return nil, fmt.Errorf("robot is blacklisted: %s", uuid)
	}

	// Verify the signature over the payload.
	// VerifyRobotSignature expects hex-encoded data (for AUTH nonces), so we
	// hex-encode the raw JSON bytes so they round-trip correctly.
	payloadHex := hex.EncodeToString([]byte(payloadJSON))
	if err := VerifyRobotSignature(robot.PublicKey, payloadHex, signature); err != nil {
		return nil, fmt.Errorf("invalid heartbeat signature for %s: %w", uuid, err)
	}

	// Parse the payload (raw JSON text)
	var payload HeartbeatPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse heartbeat payload: %w", err)
	}

	// Check sequence number (must be greater than last seen)
	existing, _ := rds.GetHeartbeat(ctx, uuid)
	if existing != nil && payload.Seq <= existing.LastSeq {
		return nil, fmt.Errorf("stale heartbeat sequence for %s: got %d, last was %d", uuid, payload.Seq, existing.LastSeq)
	}

	// Determine TTL, capped to prevent misbehaving robots from pinning Redis state.
	ttl := shared.AppConfig.Database.Redis.TTL()
	if payload.TTL > 0 {
		requested := time.Duration(payload.TTL) * time.Second
		if requested > MaxHeartbeatTTL {
			requested = MaxHeartbeatTTL
		}
		ttl = requested
	}

	// Update heartbeat state in Redis
	state := &database.HeartbeatState{
		UUID:     uuid,
		IP:       ip,
		LastSeq:  payload.Seq,
		LastSeen: time.Now().Unix(),
	}
	if err := rds.SetHeartbeat(ctx, state, ttl); err != nil {
		return nil, fmt.Errorf("failed to store heartbeat: %w", err)
	}

	// Also refresh the active robot session if one exists
	if active, _ := rds.GetActiveRobot(ctx, uuid); active != nil {
		active.IP = ip
		if err := rds.SetActiveRobot(ctx, active, ttl); err != nil {
			shared.DebugPrint("Failed to refresh active session for %s: %v", uuid, err)
		}
	}

	return &HeartbeatResult{
		UUID:    uuid,
		IP:      ip,
		Payload: &payload,
	}, nil
}

