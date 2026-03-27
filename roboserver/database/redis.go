package database

import (
	"context"
	"encoding/json"
	"fmt"
	"roboserver/shared"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisHandler struct {
	Client *redis.Client
}

func NewRedisHandler(ctx context.Context) (*RedisHandler, error) {
	cfg := shared.AppConfig.Database.Redis

	shared.DebugPrint("Connecting to Redis at %s", cfg.Addr())

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	shared.DebugPrint("Successfully connected to Redis")
	return &RedisHandler{Client: client}, nil
}

func (h *RedisHandler) Close() {
	if h.Client != nil {
		h.Client.Close()
	}
}

func (h *RedisHandler) IsHealthy(ctx context.Context) bool {
	if h.Client == nil {
		return false
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return h.Client.Ping(pingCtx).Err() == nil
}

// --- Active Robot Session Management ---

// ActiveRobot represents the ephemeral state of a connected robot in Redis.
type ActiveRobot struct {
	UUID       string `json:"uuid"`
	IP         string `json:"ip"`
	DeviceType string `json:"device_type"`
	SessionJWT string `json:"session_jwt"`
	PID        int    `json:"pid,omitempty"`
	ConnectedAt int64 `json:"connected_at"`
}

func robotKey(uuid string) string {
	return fmt.Sprintf("robot:%s:active", uuid)
}

// SetActiveRobot stores a robot's active session in Redis with TTL.
func (h *RedisHandler) SetActiveRobot(ctx context.Context, robot *ActiveRobot, ttl time.Duration) error {
	data, err := json.Marshal(robot)
	if err != nil {
		return fmt.Errorf("failed to marshal active robot: %w", err)
	}
	return h.Client.Set(ctx, robotKey(robot.UUID), data, ttl).Err()
}

// GetActiveRobot retrieves a robot's active session from Redis.
func (h *RedisHandler) GetActiveRobot(ctx context.Context, uuid string) (*ActiveRobot, error) {
	data, err := h.Client.Get(ctx, robotKey(uuid)).Bytes()
	if err != nil {
		return nil, err
	}
	r := &ActiveRobot{}
	if err := json.Unmarshal(data, r); err != nil {
		return nil, err
	}
	return r, nil
}

// RefreshHeartbeat resets the TTL on a robot's active session key.
func (h *RedisHandler) RefreshHeartbeat(ctx context.Context, uuid string, ttl time.Duration) error {
	return h.Client.Expire(ctx, robotKey(uuid), ttl).Err()
}

// RemoveActiveRobot deletes a robot's active session from Redis.
func (h *RedisHandler) RemoveActiveRobot(ctx context.Context, uuid string) error {
	return h.Client.Del(ctx, robotKey(uuid)).Err()
}

// IsRobotActive checks if a robot has an active session in Redis.
func (h *RedisHandler) IsRobotActive(ctx context.Context, uuid string) (bool, error) {
	n, err := h.Client.Exists(ctx, robotKey(uuid)).Result()
	return n > 0, err
}

// GetAllActiveRobots returns all robots with active sessions.
func (h *RedisHandler) GetAllActiveRobots(ctx context.Context) ([]*ActiveRobot, error) {
	keys, err := h.Client.Keys(ctx, "robot:*:active").Result()
	if err != nil {
		return nil, err
	}
	robots := make([]*ActiveRobot, 0, len(keys))
	for _, key := range keys {
		data, err := h.Client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		r := &ActiveRobot{}
		if err := json.Unmarshal(data, r); err != nil {
			continue
		}
		robots = append(robots, r)
	}
	return robots, nil
}

// --- Robot Public Key Storage (for PERSIST flow) ---

func robotPublicKeyKey(uuid string) string {
	return fmt.Sprintf("robot:%s:pubkey", uuid)
}

// SetRobotPublicKey stores a robot's public key in Redis alongside its session.
func (h *RedisHandler) SetRobotPublicKey(ctx context.Context, uuid, publicKey string, ttl time.Duration) error {
	return h.Client.Set(ctx, robotPublicKeyKey(uuid), publicKey, ttl).Err()
}

// GetRobotPublicKey retrieves a robot's public key from Redis.
func (h *RedisHandler) GetRobotPublicKey(ctx context.Context, uuid string) (string, error) {
	return h.Client.Get(ctx, robotPublicKeyKey(uuid)).Result()
}

// RemoveRobotPublicKey deletes a robot's stored public key.
func (h *RedisHandler) RemoveRobotPublicKey(ctx context.Context, uuid string) error {
	return h.Client.Del(ctx, robotPublicKeyKey(uuid)).Err()
}

// --- Pending Robot Registration Management ---

// PendingRobot represents a robot awaiting user approval.
type PendingRobot struct {
	UUID       string `json:"uuid"`
	IP         string `json:"ip"`
	DeviceType string `json:"device_type"`
	PublicKey  string `json:"public_key"`
	RequestedAt int64 `json:"requested_at"`
}

func pendingKey(uuid string) string {
	return fmt.Sprintf("robot:%s:pending", uuid)
}

func registrationResponseChannel(uuid string) string {
	return fmt.Sprintf("robot:%s:reg_response", uuid)
}

// SetPendingRobot stores a pending registration in Redis with TTL.
func (h *RedisHandler) SetPendingRobot(ctx context.Context, robot *PendingRobot, ttl time.Duration) error {
	// Check for duplicate UUID in both pending and active
	exists, _ := h.Client.Exists(ctx, pendingKey(robot.UUID)).Result()
	if exists > 0 {
		return fmt.Errorf("robot %s already has a pending registration", robot.UUID)
	}
	active, _ := h.Client.Exists(ctx, robotKey(robot.UUID)).Result()
	if active > 0 {
		return fmt.Errorf("robot %s is already active", robot.UUID)
	}

	data, err := json.Marshal(robot)
	if err != nil {
		return fmt.Errorf("failed to marshal pending robot: %w", err)
	}
	return h.Client.Set(ctx, pendingKey(robot.UUID), data, ttl).Err()
}

// GetPendingRobot retrieves a pending registration from Redis.
func (h *RedisHandler) GetPendingRobot(ctx context.Context, uuid string) (*PendingRobot, error) {
	data, err := h.Client.Get(ctx, pendingKey(uuid)).Bytes()
	if err != nil {
		return nil, err
	}
	r := &PendingRobot{}
	if err := json.Unmarshal(data, r); err != nil {
		return nil, err
	}
	return r, nil
}

// RemovePendingRobot deletes a pending registration from Redis.
func (h *RedisHandler) RemovePendingRobot(ctx context.Context, uuid string) error {
	return h.Client.Del(ctx, pendingKey(uuid)).Err()
}

// GetAllPendingRobots returns all robots with pending registrations.
func (h *RedisHandler) GetAllPendingRobots(ctx context.Context) ([]*PendingRobot, error) {
	keys, err := h.Client.Keys(ctx, "robot:*:pending").Result()
	if err != nil {
		return nil, err
	}
	robots := make([]*PendingRobot, 0, len(keys))
	for _, key := range keys {
		data, err := h.Client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		r := &PendingRobot{}
		if err := json.Unmarshal(data, r); err != nil {
			continue
		}
		robots = append(robots, r)
	}
	return robots, nil
}

// PublishRegistrationResponse publishes an accept/reject response for a pending robot.
func (h *RedisHandler) PublishRegistrationResponse(ctx context.Context, uuid string, accepted bool) error {
	msg := "reject"
	if accepted {
		msg = "accept"
	}
	return h.Client.Publish(ctx, registrationResponseChannel(uuid), msg).Err()
}

// WaitForRegistrationResponse blocks until a registration response is received or context expires.
// Returns true if accepted, false if rejected.
func (h *RedisHandler) WaitForRegistrationResponse(ctx context.Context, uuid string) (bool, error) {
	sub := h.Client.Subscribe(ctx, registrationResponseChannel(uuid))
	defer sub.Close()

	ch := sub.Channel()
	select {
	case msg := <-ch:
		return msg.Payload == "accept", nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
