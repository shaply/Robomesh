package handler_engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"roboserver/shared"
	"testing"
)

func init() {
	shared.AppConfig = shared.Config{
		Handlers: shared.HandlersConfig{
			BasePath: "./testdata",
		},
		Database: shared.DatabaseConfig{
			Redis: shared.RedisConfig{
				SessionTTL: "60s",
			},
		},
	}
}

func TestResolveHandlerScript(t *testing.T) {
	// Create test handler with new directory structure
	os.MkdirAll(filepath.Join("testdata", "test_robot"), 0o755)
	testScript := filepath.Join("testdata", "test_robot", "start_handler.sh")
	os.WriteFile(testScript, []byte("#!/bin/bash\necho ok"), 0o755)
	defer os.RemoveAll("testdata")

	path, err := ResolveHandlerScript("test_robot")
	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}
	if path == "" {
		t.Fatal("Resolved path is empty")
	}

	// Non-existent type should fail
	_, err = ResolveHandlerScript("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent handler")
	}
}

func TestJSONRPCEnvelopeSerialization(t *testing.T) {
	env := JSONRPCEnvelope{
		ID:     "req-1",
		Target: TargetDatabase,
		Method: "get_robot",
		Data:   "uuid-123",
	}

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded JSONRPCEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != "req-1" {
		t.Errorf("Expected ID=req-1, got %s", decoded.ID)
	}
	if decoded.Target != TargetDatabase {
		t.Errorf("Expected target=database, got %s", decoded.Target)
	}
}

func TestConnectMessageSerialization(t *testing.T) {
	msg := ConnectMessage{
		Type:       MsgTypeConnect,
		UUID:       "robot-1",
		DeviceType: "sensor",
		IP:         "192.168.1.1",
		SessionID:  "sess_abc",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ConnectMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.UUID != "robot-1" {
		t.Errorf("Expected UUID=robot-1, got %s", decoded.UUID)
	}
	if decoded.Type != MsgTypeConnect {
		t.Errorf("Expected type=connect, got %s", decoded.Type)
	}
}

func TestDisconnectMessageSerialization(t *testing.T) {
	msg := DisconnectMessage{
		Type:   MsgTypeDisconnect,
		UUID:   "robot-1",
		Reason: "heartbeat_expired",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DisconnectMessage
	json.Unmarshal(data, &decoded)
	if decoded.Reason != "heartbeat_expired" {
		t.Errorf("Expected reason=heartbeat_expired, got %s", decoded.Reason)
	}
}
