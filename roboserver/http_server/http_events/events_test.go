package http_events

import (
	"encoding/json"
	"roboserver/shared"
	"testing"
)

func TestSentEventSerialization(t *testing.T) {
	event := SentEvent{
		Id:   "1",
		Type: "robot.connected",
		Data: `{"uuid":"robot-001"}`,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal SentEvent: %v", err)
	}

	var decoded SentEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SentEvent: %v", err)
	}

	if decoded.Id != "1" {
		t.Errorf("Expected Id '1', got %q", decoded.Id)
	}
	if decoded.Type != "robot.connected" {
		t.Errorf("Expected Type 'robot.connected', got %q", decoded.Type)
	}
	if decoded.Data != `{"uuid":"robot-001"}` {
		t.Errorf("Expected Data, got %q", decoded.Data)
	}
}

func TestEventSessionSerialization(t *testing.T) {
	sess := EventSession{
		Session: shared.Session{
			UserID:    "admin",
			SessionID: "sess-123",
		},
		Timestamp: 1234567890,
		RandomID:  "abc-xyz",
	}

	data, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("Failed to marshal EventSession: %v", err)
	}

	var decoded EventSession
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal EventSession: %v", err)
	}

	if decoded.Session.UserID != "admin" {
		t.Errorf("Expected UserID 'admin', got %q", decoded.Session.UserID)
	}
	if decoded.Timestamp != 1234567890 {
		t.Errorf("Expected Timestamp 1234567890, got %d", decoded.Timestamp)
	}
	if decoded.RandomID != "abc-xyz" {
		t.Errorf("Expected RandomID 'abc-xyz', got %q", decoded.RandomID)
	}
}

func TestNewEventsManager(t *testing.T) {
	em := NewEventsManager(nil)
	if em == nil {
		t.Fatal("Expected non-nil EventsManager")
	}
	if em.bus != nil {
		t.Error("Expected nil bus")
	}
}

func TestGetClient_NotFound(t *testing.T) {
	em := NewEventsManager(nil)
	sess := &EventSession{
		Session: shared.Session{
			UserID:    "admin",
			SessionID: "nonexistent",
		},
	}
	client, ok := em.GetClient(sess)
	if ok {
		t.Error("Expected not found for nonexistent client")
	}
	if client != nil {
		t.Error("Expected nil client for nonexistent session")
	}
}

func TestEventStruct(t *testing.T) {
	es := EventStruct{
		ESess: EventSession{
			Session: shared.Session{UserID: "admin"},
		},
		EventTypes: []string{"robot.connected", "robot.disconnected"},
	}

	data, err := json.Marshal(es)
	if err != nil {
		t.Fatalf("Failed to marshal EventStruct: %v", err)
	}

	var decoded EventStruct
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal EventStruct: %v", err)
	}

	if len(decoded.EventTypes) != 2 {
		t.Errorf("Expected 2 event types, got %d", len(decoded.EventTypes))
	}
}
