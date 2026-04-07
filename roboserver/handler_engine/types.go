package handler_engine

// JSONRPCEnvelope is the standard message format between the Go sidecar and handler scripts.
type JSONRPCEnvelope struct {
	ID     string      `json:"id,omitempty"`     // Correlation ID for request-response
	Target string      `json:"target"`           // "database", "robot", "event_bus", "response"
	Method string      `json:"method,omitempty"` // Target-specific method
	Data   interface{} `json:"data,omitempty"`   // Payload
	Error  string      `json:"error,omitempty"`  // Error message (responses only)
}

// Targets for JSON-RPC routing
const (
	TargetDatabase = "database"
	TargetRobot    = "robot"
	TargetEventBus = "event_bus"
	TargetResponse = "response"
	TargetConfig   = "config"
	TargetConnect  = "connect_robot"
)

// System messages sent by the Go sidecar to handler scripts
const (
	MsgTypeConnect    = "connect"
	MsgTypeDisconnect = "disconnect"
	MsgTypeIncoming   = "incoming"
	MsgTypeEvent      = "event"
	MsgTypeHeartbeat  = "heartbeat"
)

// ConnectMessage is sent to the handler script when a robot authenticates.
type ConnectMessage struct {
	Type       string `json:"type"`
	UUID       string `json:"uuid"`
	DeviceType string `json:"device_type"`
	IP         string `json:"ip"`
	SessionID  string `json:"session_id"`
}

// DisconnectMessage is sent to the handler script when a robot disconnects.
type DisconnectMessage struct {
	Type   string `json:"type"`
	UUID   string `json:"uuid"`
	Reason string `json:"reason"`
}

// IncomingMessage wraps a message from the robot to the handler.
type IncomingMessage struct {
	Type    string `json:"type"`
	UUID    string `json:"uuid"`
	Payload string `json:"payload"`
}

// EventMessage wraps a comm bus event forwarded to the handler.
type EventMessage struct {
	Type      string      `json:"type"`
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}
