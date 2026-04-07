package http_websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"roboserver/comms"
	"roboserver/shared"
	"roboserver/shared/event_bus"
)

func init() {
	shared.AppConfig.Server.AllowedOrigins = []string{"http://localhost"}
}

func newTestBus() comms.Bus {
	eb := event_bus.NewEventBus()
	return comms.NewLocalBus(eb, nil)
}

func TestWebSocketManager_Connect(t *testing.T) {
	bus := newTestBus()
	manager := NewManager(bus)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleConnection(w, r)
	}))
	defer server.Close()

	// Convert HTTP URL to WS URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, http.Header{"Origin": []string{"http://localhost"}})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("Expected 101, got %d", resp.StatusCode)
	}
}

func TestWebSocketManager_SubscribeAndReceive(t *testing.T) {
	bus := newTestBus()
	manager := NewManager(bus)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleConnection(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, http.Header{"Origin": []string{"http://localhost"}})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Subscribe to an event
	subMsg, _ := json.Marshal(IncomingMessage{
		Action: "subscribe",
		Event:  "test.topic",
	})
	conn.WriteMessage(websocket.TextMessage, subMsg)

	// Read the ack
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read ack: %v", err)
	}

	var ack OutgoingMessage
	json.Unmarshal(msg, &ack)
	if ack.Type != "ack" {
		t.Errorf("Expected ack, got %s", ack.Type)
	}

	// Publish an event on the bus
	bus.PublishEvent("test.topic", "hello world")

	// Read the event
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read event: %v", err)
	}

	var event OutgoingMessage
	json.Unmarshal(msg, &event)
	if event.Type != "event" {
		t.Errorf("Expected event, got %s", event.Type)
	}
	if event.Event != "test.topic" {
		t.Errorf("Expected test.topic, got %s", event.Event)
	}
}

func TestWebSocketManager_InvalidAction(t *testing.T) {
	bus := newTestBus()
	manager := NewManager(bus)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manager.HandleConnection(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, http.Header{"Origin": []string{"http://localhost"}})
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send an unknown action
	msg, _ := json.Marshal(IncomingMessage{Action: "bogus"})
	conn.WriteMessage(websocket.TextMessage, msg)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, resp, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	var out OutgoingMessage
	json.Unmarshal(resp, &out)
	if out.Type != "error" {
		t.Errorf("Expected error response, got %s", out.Type)
	}
}
