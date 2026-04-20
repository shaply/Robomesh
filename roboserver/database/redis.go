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
	var robots []*ActiveRobot
	iter := h.Client.Scan(ctx, 0, "robot:*:active", 100).Iterator()
	for iter.Next(ctx) {
		data, err := h.Client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		r := &ActiveRobot{}
		if err := json.Unmarshal(data, r); err != nil {
			continue
		}
		robots = append(robots, r)
	}
	if err := iter.Err(); err != nil {
		return nil, err
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
	exists, err := h.Client.Exists(ctx, pendingKey(robot.UUID)).Result()
	if err != nil {
		return fmt.Errorf("failed to check pending status for %s: %w", robot.UUID, err)
	}
	if exists > 0 {
		return fmt.Errorf("robot %s already has a pending registration", robot.UUID)
	}
	active, err := h.Client.Exists(ctx, robotKey(robot.UUID)).Result()
	if err != nil {
		return fmt.Errorf("failed to check active status for %s: %w", robot.UUID, err)
	}
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
	var robots []*PendingRobot
	iter := h.Client.Scan(ctx, 0, "robot:*:pending", 100).Iterator()
	for iter.Next(ctx) {
		data, err := h.Client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		r := &PendingRobot{}
		if err := json.Unmarshal(data, r); err != nil {
			continue
		}
		robots = append(robots, r)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return robots, nil
}

// --- Heartbeat Tracking ---

// HeartbeatState represents a robot's heartbeat state in Redis, independent of handler sessions.
type HeartbeatState struct {
	UUID     string `json:"uuid"`
	IP       string `json:"ip"`
	LastSeq  int64  `json:"last_seq"`
	LastSeen int64  `json:"last_seen"`
}

func heartbeatKey(uuid string) string {
	return fmt.Sprintf("robot:%s:heartbeat", uuid)
}

// SetHeartbeat stores or updates a robot's heartbeat state in Redis.
func (h *RedisHandler) SetHeartbeat(ctx context.Context, state *HeartbeatState, ttl time.Duration) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat state: %w", err)
	}
	return h.Client.Set(ctx, heartbeatKey(state.UUID), data, ttl).Err()
}

// GetHeartbeat retrieves a robot's heartbeat state from Redis.
func (h *RedisHandler) GetHeartbeat(ctx context.Context, uuid string) (*HeartbeatState, error) {
	data, err := h.Client.Get(ctx, heartbeatKey(uuid)).Bytes()
	if err != nil {
		return nil, err
	}
	s := &HeartbeatState{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	return s, nil
}

// RemoveHeartbeat deletes a robot's heartbeat state from Redis.
func (h *RedisHandler) RemoveHeartbeat(ctx context.Context, uuid string) error {
	return h.Client.Del(ctx, heartbeatKey(uuid)).Err()
}

// IsRobotOnline checks if a robot has a current heartbeat (independent of handler).
func (h *RedisHandler) IsRobotOnline(ctx context.Context, uuid string) (bool, error) {
	n, err := h.Client.Exists(ctx, heartbeatKey(uuid)).Result()
	return n > 0, err
}

// GetAllOnlineRobots returns all robots with active heartbeats.
func (h *RedisHandler) GetAllOnlineRobots(ctx context.Context) ([]*HeartbeatState, error) {
	var states []*HeartbeatState
	iter := h.Client.Scan(ctx, 0, "robot:*:heartbeat", 100).Iterator()
	for iter.Next(ctx) {
		data, err := h.Client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		s := &HeartbeatState{}
		if err := json.Unmarshal(data, s); err != nil {
			continue
		}
		states = append(states, s)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return states, nil
}

// --- User Authentication ---

// User represents a user account stored in Redis.
type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

func userKey(username string) string {
	return fmt.Sprintf("user:%s", username)
}

// SetUser stores a user in Redis (no TTL — permanent until deleted).
func (h *RedisHandler) SetUser(ctx context.Context, user *User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}
	return h.Client.Set(ctx, userKey(user.Username), data, 0).Err()
}

// GetUser retrieves a user from Redis by username.
func (h *RedisHandler) GetUser(ctx context.Context, username string) (*User, error) {
	data, err := h.Client.Get(ctx, userKey(username)).Bytes()
	if err != nil {
		return nil, err
	}
	u := &User{}
	if err := json.Unmarshal(data, u); err != nil {
		return nil, err
	}
	return u, nil
}

// --- User Session Management ---

func userSessionKey(token string) string {
	return fmt.Sprintf("session:%s", token)
}

// SetUserSession stores a user session token in Redis with TTL.
func (h *RedisHandler) SetUserSession(ctx context.Context, token, username string, ttl time.Duration) error {
	return h.Client.Set(ctx, userSessionKey(token), username, ttl).Err()
}

// GetUserSession retrieves the username associated with a session token.
func (h *RedisHandler) GetUserSession(ctx context.Context, token string) (string, error) {
	return h.Client.Get(ctx, userSessionKey(token)).Result()
}

// RemoveUserSession deletes a user session from Redis.
func (h *RedisHandler) RemoveUserSession(ctx context.Context, token string) error {
	return h.Client.Del(ctx, userSessionKey(token)).Err()
}

// --- SSE Ticket Management ---

func ticketKey(ticket string) string {
	return fmt.Sprintf("ticket:%s", ticket)
}

// SetTicket stores a single-use SSE ticket in Redis with a short TTL.
func (h *RedisHandler) SetTicket(ctx context.Context, ticket, username string, ttl time.Duration) error {
	return h.Client.Set(ctx, ticketKey(ticket), username, ttl).Err()
}

// ConsumeTicket retrieves and deletes a ticket atomically (single-use).
// Returns the username associated with the ticket, or error if not found/expired.
func (h *RedisHandler) ConsumeTicket(ctx context.Context, ticket string) (string, error) {
	key := ticketKey(ticket)
	username, err := h.Client.GetDel(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return username, nil
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
	case msg, ok := <-ch:
		if !ok {
			// Channel closed (e.g. Redis disconnected) — surface as error
			// instead of silently returning "rejected".
			return false, fmt.Errorf("registration response channel closed unexpectedly")
		}
		return msg.Payload == "accept", nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
