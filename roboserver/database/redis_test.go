package database

import (
	"encoding/json"
	"testing"
)

func TestActiveRobotSerialization(t *testing.T) {
	robot := &ActiveRobot{
		UUID:        "robot-001",
		IP:          "192.168.1.50",
		DeviceType:  "example_robot",
		SessionJWT:  "jwt-token-abc",
		PID:         12345,
		ConnectedAt: 1711234567,
	}

	data, err := json.Marshal(robot)
	if err != nil {
		t.Fatalf("Failed to marshal ActiveRobot: %v", err)
	}

	var decoded ActiveRobot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ActiveRobot: %v", err)
	}

	if decoded.UUID != "robot-001" {
		t.Errorf("Expected UUID=robot-001, got %s", decoded.UUID)
	}
	if decoded.IP != "192.168.1.50" {
		t.Errorf("Expected IP=192.168.1.50, got %s", decoded.IP)
	}
	if decoded.DeviceType != "example_robot" {
		t.Errorf("Expected DeviceType=example_robot, got %s", decoded.DeviceType)
	}
	if decoded.PID != 12345 {
		t.Errorf("Expected PID=12345, got %d", decoded.PID)
	}
	if decoded.ConnectedAt != 1711234567 {
		t.Errorf("Expected ConnectedAt=1711234567, got %d", decoded.ConnectedAt)
	}
}

func TestActiveRobotPIDOmitempty(t *testing.T) {
	robot := &ActiveRobot{
		UUID:        "robot-002",
		IP:          "10.0.0.1",
		DeviceType:  "sensor",
		SessionJWT:  "jwt",
		ConnectedAt: 1234567890,
	}

	data, err := json.Marshal(robot)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// PID=0 should be omitted due to omitempty
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	if _, exists := raw["pid"]; exists {
		t.Error("PID=0 should be omitted from JSON")
	}
}

func TestPendingRobotSerialization(t *testing.T) {
	robot := &PendingRobot{
		UUID:        "pending-001",
		IP:          "10.0.0.5",
		DeviceType:  "rover",
		PublicKey:   "aabbccdd11223344aabbccdd11223344aabbccdd11223344aabbccdd11223344",
		RequestedAt: 1711234567,
	}

	data, err := json.Marshal(robot)
	if err != nil {
		t.Fatalf("Failed to marshal PendingRobot: %v", err)
	}

	var decoded PendingRobot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PendingRobot: %v", err)
	}

	if decoded.UUID != "pending-001" {
		t.Errorf("Expected UUID=pending-001, got %s", decoded.UUID)
	}
	if decoded.DeviceType != "rover" {
		t.Errorf("Expected DeviceType=rover, got %s", decoded.DeviceType)
	}
	if decoded.PublicKey != robot.PublicKey {
		t.Errorf("PublicKey mismatch")
	}
	if decoded.RequestedAt != 1711234567 {
		t.Errorf("Expected RequestedAt=1711234567, got %d", decoded.RequestedAt)
	}
}

func TestRedisKeyFormats(t *testing.T) {
	// Active robot key
	key := robotKey("robot-123")
	if key != "robot:robot-123:active" {
		t.Errorf("Expected robot:robot-123:active, got %s", key)
	}

	// Pending robot key
	pKey := pendingKey("robot-456")
	if pKey != "robot:robot-456:pending" {
		t.Errorf("Expected robot:robot-456:pending, got %s", pKey)
	}

	// Public key storage key
	pkKey := robotPublicKeyKey("robot-789")
	if pkKey != "robot:robot-789:pubkey" {
		t.Errorf("Expected robot:robot-789:pubkey, got %s", pkKey)
	}

	// Registration response channel
	ch := registrationResponseChannel("robot-abc")
	if ch != "robot:robot-abc:reg_response" {
		t.Errorf("Expected robot:robot-abc:reg_response, got %s", ch)
	}
}

func TestRedisKeyUniqueness(t *testing.T) {
	uuid := "same-robot"
	activeKey := robotKey(uuid)
	pndKey := pendingKey(uuid)
	pubKeyKey := robotPublicKeyKey(uuid)
	respCh := registrationResponseChannel(uuid)

	keys := map[string]bool{
		activeKey: true,
		pndKey:    true,
		pubKeyKey: true,
		respCh:    true,
	}

	if len(keys) != 4 {
		t.Error("Redis keys for the same UUID should all be unique")
	}
}
