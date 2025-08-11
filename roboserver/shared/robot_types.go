// Package shared contains core types and interfaces for the Robomesh server system.
//
// This package defines the fundamental abstractions for robot management:
// - Robot: Core robot state and behavior interface
// - RobotHandler: Communication and state management
// - RobotConnHandler: Connection lifecycle management
// - Msg: Inter-component message passing
//
// The design uses composition over inheritance, allowing specific robot types
// to embed BaseRobot while adding their own specialized fields and behaviors.
package shared

// RobotType represents the category of robot and determines its capabilities.
// Used by the factory pattern to create appropriate handlers for different robot types.
type RobotType string

// BASE_ROBOT_TYPE is the default robot type for generic robots without specialized functionality.
const BASE_ROBOT_TYPE RobotType = "base_robot"

// Robot defines the core interface that all robot types must implement.
//
// This interface provides a consistent API for robot state management and serialization
// across different robot implementations. Robot types should embed BaseRobot and implement
// this interface to ensure compatibility with the robot management system.
//
// Design Pattern:
// Robot types use composition by embedding BaseRobot and extending functionality:
//
//	type TrashRobot struct {
//	    BaseRobot
//	    BinCapacity int
//	    CurrentLoad int
//	}
//
// Usage with Type Assertion:
//
//	robot := manager.GetRobot("device_id", "")
//	if trashRobot, ok := robot.(*TrashRobot); ok {
//	    fmt.Printf("Bin load: %d/%d\n", trashRobot.CurrentLoad, trashRobot.BinCapacity)
//	}
//
// All methods should be thread-safe when called on the same robot instance.
type Robot interface {
	ToJSON() string              // Serialize robot state to JSON string for API responses
	GetBaseRobot() BaseRobot     // Access embedded base robot structure
	GetDeviceID() string         // Get unique device identifier
	GetIP() string               // Get current IP address
	IsOnline() bool              // Check if robot is currently connected and responsive
	SetLastSeen(timestamp int64) // Update last activity timestamp (Unix timestamp)
	String() string              // Human-readable representation for logging/debugging
}

// BaseRobot provides the fundamental state and metadata for all robot types.
//
// This struct contains common fields that every robot needs, regardless of its specialized
// functionality. Robot implementations should embed this struct and add their own fields.
//
// Field Details:
// - DeviceID: Persistent unique identifier, used for authentication and tracking
// - IP: Current network address, may change during robot's lifetime
// - RobotType: Determines which factory function creates the robot's handlers
// - Status: Current operational state (e.g., "online", "offline", "busy", "error")
// - Battery: Optional power level (0-100), omitted if not applicable
// - LastSeen: Unix timestamp of last communication, used for health monitoring
// - AuthToken: Credentials for secure communication, never serialized to JSON
//
// JSON Serialization:
// Fields with `omitempty` are excluded when zero-valued.
// AuthToken is never serialized due to `json:"-"` tag.
//
// Thread Safety:
// Individual field access is not synchronized. Callers should use appropriate
// locking when modifying fields that may be accessed concurrently.
type BaseRobot struct {
	DeviceID  string    `json:"device_id"`           // Unique identifier for robot authentication and tracking
	IP        string    `json:"ip,omitempty"`        // Current IP address for network communication
	RobotType RobotType `json:"robot_type"`          // Robot category determining capabilities and handlers
	Status    string    `json:"status"`              // Current operational state
	Battery   byte      `json:"battery,omitempty"`   // Power level (0-100), omitted if not applicable
	LastSeen  int64     `json:"last_seen,omitempty"` // Unix timestamp of last communication
	AuthToken string    `json:"-"`                   // Security credentials, never serialized
}

// BaseRobotHandler provides a default implementation of the RobotHandler interface.
//
// This struct serves as a foundation for robot-specific handlers and contains
// the essential components for robot communication and state management.
//
// Components:
// - Robot: The actual robot state implementing the Robot interface
// - MsgChan: Buffered channel for queuing incoming messages
// - disconnect: Signal channel for coordinating shutdown
//
// Usage:
// Robot implementations can embed this struct and override methods as needed.
// The message channel should be appropriately sized based on expected message volume.
type BaseRobotHandler struct {
	Robot      Robot     `json:"-"` // Robot state and behavior implementation
	MsgChan    chan Msg  `json:"-"` // Buffered message queue for asynchronous communication
	disconnect chan bool `json:"-"` // Coordination channel for graceful shutdown
}

// RobotHandler defines the interface for managing robot state and communication.
//
// This interface abstracts robot communication and provides a consistent API
// for different robot types. Handlers are responsible for:
// - Maintaining robot state
// - Processing incoming messages
// - Providing access to robot metadata
// - Coordinating graceful shutdown
//
// Message Processing:
// Messages are queued asynchronously in the robot's message channel.
// Implementations should define appropriate queue sizes based on expected load.
//
// Lifecycle:
// Handlers are created by RobotConnHandler instances and managed by RobotManager.
// The disconnect channel coordinates cleanup when robots disconnect.
//
// Thread Safety:
// Implementations should be thread-safe for concurrent access from multiple goroutines.
type RobotHandler interface {
	GetRobot() Robot                 // Access robot state for API responses and status checks
	SendMsg(msg Msg) error           // Queue message for asynchronous processing by robot
	GetDeviceID() string             // Get unique robot identifier for routing and logging
	GetIP() string                   // Get current IP address for network diagnostics
	GetDisconnectChannel() chan bool // Get coordination channel for graceful shutdown
	QuickAction()                    // Perform immediate status check or health ping
}

// Msg defines the interface for inter-component message passing.
//
// Messages enable asynchronous communication between system components and robots.
// The interface supports various message patterns:
// - Fire-and-forget: Messages without reply channels
// - Request-response: Messages with reply channels for synchronous-style communication
// - Broadcast: Messages sent to multiple recipients
//
// Message Flow:
// 1. Sender creates message with appropriate payload and source
// 2. Message is queued in target robot's channel
// 3. Robot processes message asynchronously
// 4. Optional reply sent through reply channel
//
// Thread Safety:
// Message implementations should be safe for concurrent access.
// Reply channels should be buffered to prevent blocking.
type Msg interface {
	GetMsg() string         // Get primary message content/command
	GetPayload() any        // Get structured data payload (optional)
	GetSource() string      // Get originating component identifier
	GetReplyChan() chan any // Get reply channel for response (optional)
}

// DefaultMsg provides a standard implementation of the Msg interface.
//
// This implementation covers most use cases for system messages:
// - Simple commands (Msg field only)
// - Commands with data (Msg + Payload)
// - Trackable messages (includes Source)
// - Request-response patterns (includes ReplyChan)
//
// JSON Serialization:
// The struct can be serialized for network transmission, but ReplyChan
// is excluded since channels cannot be serialized.
//
// Example Usage:
//
//	// Simple command
//	msg := DefaultMsg{Msg: "STATUS_CHECK", Source: "health_monitor"}
//
//	// Command with response
//	replyChan := make(chan any, 1)
//	msg := DefaultMsg{
//	    Msg: "GET_BATTERY",
//	    Source: "dashboard",
//	    ReplyChan: replyChan,
//	}
type DefaultMsg struct {
	Msg       string   `json:"msg"`               // Primary command or message type
	Payload   any      `json:"payload,omitempty"` // Structured data payload (optional)
	Source    string   `json:"source,omitempty"`  // Originating component for tracing
	ReplyChan chan any `json:"-"`                 // Response channel, not serialized
}

// NewRobotConnHandlerFunc defines the factory function signature for creating robot connection handlers.
//
// This function type enables the factory pattern for robot creation:
// - Different robot types register their own constructor functions
// - RobotManager uses these functions to create appropriate handlers
// - Automatic registration via init() functions in robot packages
//
// Parameters:
//   - deviceId: Unique robot identifier for tracking and authentication
//   - ip: Robot's network address for communication setup
//
// Returns:
//   - RobotConnHandler: Configured handler ready to manage robot lifecycle
//   - error: Creation failure (invalid parameters, resource constraints, etc.)
//
// Implementation Requirements:
// Factory functions should:
// - Validate input parameters
// - Initialize robot state with provided metadata
// - Set up communication channels with appropriate buffer sizes
// - Return configured handler ready for immediate use
//
// Example Registration:
//
//	func init() {
//	    shared.RegisterRobotType("trash", NewTrashRobotConnHandler)
//	}
type NewRobotConnHandlerFunc func(deviceId string, ip string) (RobotConnHandler, error)

// BaseRobotConnHandler provides a foundation for robot-specific connection handlers.
//
// This struct contains the common components needed for robot connection management
// and can be embedded by specific robot type implementations.
//
// Components:
// - DeviceID: Robot's unique identifier
// - IP: Current network address
// - Handler: Robot state and communication manager
// - DisconnectChan: Coordination channel for graceful shutdown
//
// Lifecycle:
// 1. Created by factory function during robot registration
// 2. Managed by RobotManager for the robot's lifetime
// 3. Cleaned up when robot disconnects or server shuts down
type BaseRobotConnHandler struct {
	DeviceID       string       `json:"device_id"` // Unique robot identifier
	IP             string       `json:"ip"`        // Current network address
	Handler        RobotHandler `json:"-"`         // State and communication manager
	DisconnectChan chan bool    `json:"-"`         // Shutdown coordination channel
}

// RobotConnHandler manages the complete lifecycle of a robot connection.
//
// This interface abstracts the connection management layer, handling:
// - Connection establishment and maintenance
// - Message processing loop coordination
// - Graceful shutdown and resource cleanup
// - Integration with the robot management system
//
// Lifecycle Management:
// 1. Start(): Begins message processing and communication loops
// 2. Active Operation: Processes messages and maintains connection
// 3. Stop(): Gracefully shuts down and cleans up resources
//
// Integration:
// - GetHandler(): Provides RobotHandler for state management
// - GetDisconnectChannel(): Enables coordination with RobotManager
//
// Implementation Requirements:
// - Start() should run indefinitely until disconnection or error
// - Stop() must be safe to call multiple times
// - Resource cleanup should be thorough to prevent leaks
// - Disconnect channel should signal when connection is lost
//
// Thread Safety:
// Implementations should be safe for concurrent Start()/Stop() calls.
type RobotConnHandler interface {
	Start() error                    // Begin connection lifecycle and message processing
	Stop() error                     // Gracefully shutdown and cleanup resources
	GetHandler() RobotHandler        // Access robot state and communication interface
	GetDisconnectChannel() chan bool // Get coordination channel for connection events
}
