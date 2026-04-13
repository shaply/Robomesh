package mqtt_server

import (
	"context"
	"encoding/json"
	"fmt"
	robotauth "roboserver/auth"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/handler_engine"
	"roboserver/shared"
	"strings"
	"time"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

// MQTTServer_t holds the MQTT broker and references to shared resources.
type MQTTServer_t struct {
	server *mqtt.Server
	bus    comms.Bus
	db     database.DBManager
	ctx    context.Context
}

// Start initializes and runs the MQTT broker.
// Robots communicate via topic-based protocol:
//
//	robomesh/auth/{uuid}          — Robot publishes auth request, server responds on robomesh/auth/{uuid}/response
//	robomesh/heartbeat/{uuid}     — Robot publishes signed heartbeat
//	robomesh/message/{uuid}       — Robot publishes messages to its handler
//	robomesh/to_robot/{uuid}      — Server publishes messages to a specific robot
func Start(ctx context.Context, bus comms.Bus, db database.DBManager) error {
	port := shared.AppConfig.Server.MQTTPort

	server := mqtt.New(&mqtt.Options{
		InlineClient: true,
	})

	s := &MQTTServer_t{
		server: server,
		bus:    bus,
		db:     db,
		ctx:    ctx,
	}

	// Custom ACL hook: allows all connections (identity verified at app layer
	// via challenge-response) but restricts topic subscriptions so clients can
	// only read response topics for their own UUID.
	aclHook := &robotACLHook{}
	if err := server.AddHook(aclHook, nil); err != nil {
		return fmt.Errorf("failed to add MQTT ACL hook: %w", err)
	}

	// Add event bus bridge hook to forward MQTT publishes to the internal event bus
	if bus != nil {
		bridgeHook := &eventBusBridgeHook{bus: bus}
		if err := server.AddHook(bridgeHook, nil); err != nil {
			shared.DebugPrint("Failed to add MQTT event bus bridge: %v", err)
		}
	}

	// Add protocol hook to handle robot auth, heartbeat, and messaging
	protocolHook := &protocolHook{mqtt: s}
	if err := server.AddHook(protocolHook, nil); err != nil {
		return fmt.Errorf("failed to add MQTT protocol hook: %w", err)
	}

	// TCP listener
	tcp := listeners.NewTCP(listeners.Config{
		ID:      "mqtt-tcp",
		Address: fmt.Sprintf(":%d", port),
	})
	if err := server.AddListener(tcp); err != nil {
		return fmt.Errorf("failed to add MQTT TCP listener: %w", err)
	}

	// Subscribe to event bus for handler→robot messages and forward via MQTT
	if bus != nil {
		s.setupOutboundBridge()
	}

	// Start server
	go func() {
		shared.DebugPrint("Starting MQTT server on port %d", port)
		if err := server.Serve(); err != nil {
			shared.DebugPrint("MQTT server error: %v", err)
		}
	}()

	<-ctx.Done()
	shared.DebugPrint("Shutting down MQTT server...")
	if err := server.Close(); err != nil {
		shared.DebugPrint("Error shutting down MQTT server: %v", err)
		return fmt.Errorf("error shutting down MQTT server: %w", err)
	}
	shared.DebugPrint("MQTT server shut down gracefully")
	return nil
}

// setupOutboundBridge subscribes to handler.*.message events on the event bus
// and publishes them to the MQTT topic robomesh/to_robot/{uuid} so that
// MQTT-connected robots receive messages from their handlers.
func (s *MQTTServer_t) setupOutboundBridge() {
	s.bus.SubscribeEvent("mqtt.to_robot", func(eventType string, data any) {
		msg, ok := data.(map[string]interface{})
		if !ok {
			return
		}
		uuid, _ := msg["uuid"].(string)
		payload, _ := msg["payload"]
		if uuid == "" {
			return
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return
		}

		topic := fmt.Sprintf("robomesh/to_robot/%s", uuid)
		s.server.Publish(topic, jsonData, false, 0)
	})
}

// --- Protocol Hook ---

// protocolHook intercepts MQTT publishes on robomesh/ topics and implements
// the robot communication protocol.
type protocolHook struct {
	mqtt.HookBase
	mqtt *MQTTServer_t
}

func (h *protocolHook) ID() string {
	return "protocol-handler"
}

func (h *protocolHook) Provides(b byte) bool {
	return b == mqtt.OnPublished
}

// AuthRequest is the JSON payload a robot publishes to robomesh/auth/{uuid}.
type AuthRequest struct {
	UUID      string `json:"uuid"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce,omitempty"`
}

// AuthResponse is published back to robomesh/auth/{uuid}/response.
type AuthResponse struct {
	Status string `json:"status"` // "nonce", "ok", "error"
	Nonce  string `json:"nonce,omitempty"`
	JWT    string `json:"jwt,omitempty"`
	Error  string `json:"error,omitempty"`
}

// HeartbeatRequest is the JSON payload for robomesh/heartbeat/{uuid}.
type HeartbeatRequest struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

func (h *protocolHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	topic := pk.TopicName
	payload := pk.Payload

	switch {
	case strings.HasPrefix(topic, "robomesh/auth/"):
		uuid := strings.TrimPrefix(topic, "robomesh/auth/")
		uuid = strings.TrimSuffix(uuid, "/request")
		if uuid != "" && !strings.Contains(uuid, "/") {
			go h.handleAuth(cl, uuid, payload)
		}

	case strings.HasPrefix(topic, "robomesh/heartbeat/"):
		uuid := strings.TrimPrefix(topic, "robomesh/heartbeat/")
		if uuid != "" && !strings.Contains(uuid, "/") {
			go h.handleHeartbeat(uuid, payload, cl)
		}

	case strings.HasPrefix(topic, "robomesh/message/"):
		uuid := strings.TrimPrefix(topic, "robomesh/message/")
		if uuid != "" && !strings.Contains(uuid, "/") {
			go h.handleMessage(uuid, payload)
		}
	}
}

// handleAuth implements a two-step challenge-response auth over MQTT.
// Step 1: Robot publishes {"uuid":"..."} → server responds with {"status":"nonce","nonce":"..."}
// Step 2: Robot publishes {"uuid":"...","signature":"...","nonce":"..."} → server responds with JWT or error
func (h *protocolHook) handleAuth(cl *mqtt.Client, uuid string, payload []byte) {
	responseTopic := fmt.Sprintf("robomesh/auth/%s/response", uuid)

	db := h.mqtt.db
	if db == nil {
		h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "no database"})
		return
	}

	pg := db.Postgres()
	rds := db.Redis()
	if pg == nil || rds == nil {
		h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "database unavailable"})
		return
	}

	var req AuthRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "invalid JSON"})
		return
	}

	// Step 1: No signature → look up robot, issue a nonce, cache robot info
	if req.Signature == "" {
		robot, err := pg.GetRobotByUUID(h.mqtt.ctx, uuid)
		if err != nil {
			h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "unknown robot"})
			return
		}
		if robot.IsBlacklisted {
			h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "blacklisted"})
			return
		}

		nonce, err := robotauth.GenerateNonce()
		if err != nil {
			h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "nonce generation failed"})
			return
		}

		// Store nonce + robot info in Redis with short TTL so step 2 can
		// skip the second PostgreSQL lookup.
		const nonceTTL = 30 * time.Second
		nonceKey := fmt.Sprintf("mqtt:nonce:%s", uuid)
		authCache := fmt.Sprintf("%s|%s|%s", nonce, robot.PublicKey, robot.DeviceType)
		rds.Client.Set(h.mqtt.ctx, nonceKey, authCache, nonceTTL)

		h.publishJSON(responseTopic, AuthResponse{Status: "nonce", Nonce: nonce})
		return
	}

	// Step 2: Signature provided → retrieve cached nonce+robot, verify, issue JWT
	nonceKey := fmt.Sprintf("mqtt:nonce:%s", uuid)
	authCache, err := rds.Client.GetDel(h.mqtt.ctx, nonceKey).Result()
	if err != nil || authCache == "" {
		h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "no pending nonce (send auth request first)"})
		return
	}

	// Parse cached "nonce|publicKey|deviceType"
	parts := strings.SplitN(authCache, "|", 3)
	if len(parts) != 3 {
		h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "corrupted auth state"})
		return
	}
	nonce, publicKey, deviceType := parts[0], parts[1], parts[2]

	// Verify signature over the nonce
	if err := robotauth.VerifyRobotSignature(publicKey, nonce, req.Signature); err != nil {
		h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "signature verification failed"})
		return
	}

	// Issue JWT
	ip := cl.Net.Remote
	sessionID := robotauth.GenerateSessionID()
	jwt, err := robotauth.IssueSessionJWT(uuid, deviceType, ip, sessionID)
	if err != nil {
		h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "failed to issue JWT"})
		return
	}

	// Store active session in Redis
	ttl := shared.AppConfig.Database.Redis.TTL()
	activeRobot := &database.ActiveRobot{
		UUID:        uuid,
		IP:          ip,
		DeviceType:  deviceType,
		SessionJWT:  jwt,
		ConnectedAt: time.Now().Unix(),
	}
	if err := rds.SetActiveRobot(h.mqtt.ctx, activeRobot, ttl); err != nil {
		h.publishJSON(responseTopic, AuthResponse{Status: "error", Error: "failed to store session"})
		return
	}

	// Spawn or reattach handler
	robotSend := func(data []byte) error {
		topic := fmt.Sprintf("robomesh/to_robot/%s", uuid)
		return h.mqtt.server.Publish(topic, data, false, 0)
	}

	if existing, ok := handler_engine.HandlerManager.Get(uuid); ok {
		// Handler already running — reattach with updated MQTT send callback
		existing.Reattach(robotSend, ip, sessionID)
		shared.DebugPrint("MQTT: Robot %s reattached to existing handler (PID %d)", uuid, existing.PID)
	} else if handler_engine.HandlerManager.TryStartSpawning(uuid) {
		_, spawnErr := handler_engine.SpawnHandlerProcess(
			h.mqtt.ctx,
			uuid, deviceType, ip, sessionID,
			pg, rds, h.mqtt.bus,
			robotSend,
		)
		handler_engine.HandlerManager.FinishSpawning(uuid)
		if spawnErr != nil {
			shared.DebugPrint("MQTT: Failed to spawn handler for %s: %v", uuid, spawnErr)
		}
	}

	shared.DebugPrint("MQTT: Robot %s authenticated successfully", uuid)
	h.publishJSON(responseTopic, AuthResponse{Status: "ok", JWT: jwt})
}

// handleHeartbeat processes a signed heartbeat from an MQTT-connected robot.
func (h *protocolHook) handleHeartbeat(uuid string, payload []byte, cl *mqtt.Client) {
	db := h.mqtt.db
	if db == nil {
		return
	}

	pg := db.Postgres()
	rds := db.Redis()
	if pg == nil || rds == nil {
		return
	}

	var req HeartbeatRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		shared.DebugPrint("MQTT heartbeat: invalid JSON from %s", uuid)
		return
	}

	ip := cl.Net.Remote

	result, err := robotauth.ProcessHeartbeat(h.mqtt.ctx, uuid, req.Payload, req.Signature, ip, pg, rds)
	if err != nil {
		shared.DebugPrint("MQTT heartbeat failed for %s: %v", uuid, err)
		responseTopic := fmt.Sprintf("robomesh/heartbeat/%s/response", uuid)
		h.publishJSON(responseTopic, map[string]string{"status": "error", "error": "heartbeat rejected"})
		return
	}

	// Publish heartbeat event for handlers with forward_heartbeats enabled
	if h.mqtt.bus != nil {
		h.mqtt.bus.PublishEvent(fmt.Sprintf("robot.%s.heartbeat", result.UUID), result)
	}

	responseTopic := fmt.Sprintf("robomesh/heartbeat/%s/response", uuid)
	h.publishJSON(responseTopic, map[string]string{"status": "ok"})
}

// handleMessage forwards a message from an MQTT-connected robot to its handler.
// Verifies the robot has an active session before forwarding.
func (h *protocolHook) handleMessage(uuid string, payload []byte) {
	// Verify robot has an active session (completed auth flow)
	db := h.mqtt.db
	if db == nil {
		return
	}
	rds := db.Redis()
	if rds == nil {
		return
	}
	if active, err := rds.GetActiveRobot(h.mqtt.ctx, uuid); active == nil || err != nil {
		shared.DebugPrint("MQTT message rejected: no active session for %s", uuid)
		return
	}

	hp, ok := handler_engine.HandlerManager.Get(uuid)
	if !ok {
		shared.DebugPrint("MQTT message: no handler for %s", uuid)
		return
	}

	hp.SendIncoming(string(payload))
}

func (h *protocolHook) publishJSON(topic string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	h.mqtt.server.Publish(topic, jsonData, false, 0)
}
