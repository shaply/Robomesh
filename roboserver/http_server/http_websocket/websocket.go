package http_websocket

import (
	"encoding/json"
	"net/http"
	"roboserver/comms"
	"roboserver/shared"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		for _, allowed := range shared.AppConfig.Server.AllowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	},
}

// IncomingMessage is the JSON envelope for messages from the browser.
type IncomingMessage struct {
	Action string          `json:"action"` // "send_to_robot", "subscribe", "unsubscribe"
	UUID   string          `json:"uuid,omitempty"`
	Event  string          `json:"event,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// OutgoingMessage is the JSON envelope for messages to the browser.
type OutgoingMessage struct {
	Type  string `json:"type"` // "event", "error", "ack"
	Event string `json:"event,omitempty"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// WSClient manages a single WebSocket connection.
type WSClient struct {
	conn    *websocket.Conn
	bus     comms.Bus
	send    chan []byte
	done    chan struct{}
	closeMu sync.Once

	cancelMu    sync.Mutex
	cancelFuncs map[string]func()
}

// Manager tracks all active WebSocket clients.
type Manager struct {
	bus     comms.Bus
	clients sync.Map
}

func NewManager(bus comms.Bus) *Manager {
	return &Manager{bus: bus}
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 8192
)

// HandleConnection upgrades an HTTP request to a WebSocket connection.
func (m *Manager) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		shared.DebugPrint("WebSocket upgrade failed: %v", err)
		return
	}

	client := &WSClient{
		conn:        conn,
		bus:         m.bus,
		send:        make(chan []byte, 256),
		done:        make(chan struct{}),
		cancelFuncs: make(map[string]func()),
	}

	m.clients.Store(client, true)

	go client.writePump()
	go client.readPump(m)
}

func (c *WSClient) close(m *Manager) {
	c.closeMu.Do(func() {
		close(c.done)
		c.conn.Close()
		if m != nil {
			m.clients.Delete(c)
		}
		c.cancelMu.Lock()
		for _, cancel := range c.cancelFuncs {
			cancel()
		}
		c.cancelFuncs = nil
		c.cancelMu.Unlock()
	})
}

func (c *WSClient) readPump(m *Manager) {
	defer c.close(m)
	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				shared.DebugPrint("WebSocket read error: %v", err)
			}
			return
		}

		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendError("invalid JSON")
			continue
		}

		c.handleMessage(&msg)
	}
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *WSClient) handleMessage(msg *IncomingMessage) {
	switch msg.Action {
	case "subscribe":
		c.subscribe(msg.Event)
	case "unsubscribe":
		c.unsubscribe(msg.Event)
	default:
		c.sendError("unknown action: " + msg.Action)
	}
}

func (c *WSClient) subscribe(eventType string) {
	if eventType == "" {
		c.sendError("event type required")
		return
	}

	cancel, err := c.bus.SubscribeEvent(eventType, func(et string, data any) {
		c.sendEvent(et, data)
	})
	if err != nil {
		c.sendError("subscribe failed: " + err.Error())
		return
	}

	c.cancelMu.Lock()
	if existing, ok := c.cancelFuncs[eventType]; ok {
		existing()
	}
	c.cancelFuncs[eventType] = cancel
	c.cancelMu.Unlock()

	c.sendAck("subscribed to " + eventType)
}

func (c *WSClient) unsubscribe(eventType string) {
	c.cancelMu.Lock()
	if cancel, ok := c.cancelFuncs[eventType]; ok {
		cancel()
		delete(c.cancelFuncs, eventType)
	}
	c.cancelMu.Unlock()

	c.sendAck("unsubscribed from " + eventType)
}

func (c *WSClient) sendEvent(eventType string, data any) {
	c.sendMsg(&OutgoingMessage{Type: "event", Event: eventType, Data: data})
}

func (c *WSClient) sendError(msg string) {
	c.sendMsg(&OutgoingMessage{Type: "error", Error: msg})
}

func (c *WSClient) sendAck(msg string) {
	c.sendMsg(&OutgoingMessage{Type: "ack", Data: msg})
}

func (c *WSClient) sendMsg(msg *OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		// Channel full, drop message
	}
}
