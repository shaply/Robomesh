// Package shared provides base implementations for robot management components.
//
// This file contains constructor functions and method implementations for the base
// robot types defined in types.go. These implementations provide the fundamental
// functionality that can be embedded and extended by specific robot types.
package shared

import (
	"encoding/json"
	"fmt"
)

// Constructor Functions
//
// These functions create properly initialized instances of the base robot types.

// NewBaseRobot creates a new BaseRobot instance with the provided parameters.
//
// This constructor initializes a BaseRobot with all necessary fields. It serves
// as the foundation for all robot types in the system and can be embedded by
// specialized robot implementations.
//
// Parameters:
//   - deviceID: Unique identifier for robot authentication and tracking
//   - ip: Current IP address for network communication
//   - robotType: Category determining robot capabilities and handlers
//   - status: Current operational state (e.g., "online", "offline", "busy")
//   - battery: Power level from 0-100 (use 0 if not applicable)
//   - lastSeen: Unix timestamp of last communication (use time.Now().Unix())
//   - authToken: Security credentials for communication (can be empty for now)
//
// Returns:
//   - *BaseRobot: Properly initialized robot instance
//
// Example Usage:
//
//	robot := shared.NewBaseRobot(
//	    "sensor_001",
//	    "192.168.1.100",
//	    shared.PROXIMITY_SENSOR_TYPE,
//	    "online",
//	    85,
//	    time.Now().Unix(),
//	    "",
//	)
func NewBaseRobot(deviceID string, ip string, robotType RobotType, status string, battery byte, lastSeen int64, authToken string) *BaseRobot {
	return &BaseRobot{
		DeviceID:  deviceID,
		IP:        ip,
		RobotType: robotType,
		Status:    status,
		Battery:   battery,
		LastSeen:  lastSeen,
		AuthToken: authToken,
	}
}

// NewBaseRobotHandler creates a new BaseRobotHandler with the provided components.
//
// This constructor initializes a handler that manages robot state and communication.
// The handler serves as an intermediary between the robot state and the communication
// system, processing messages and coordinating shutdown.
//
// Parameters:
//   - robot: Robot instance implementing the Robot interface
//   - msg_chan: Buffered channel for queuing incoming messages
//   - disconnect: Channel for coordinating graceful shutdown (must not be nil)
//
// Returns:
//   - *BaseRobotHandler: Properly initialized handler instance
//
// Panics:
//   - If disconnect channel is nil (critical for proper cleanup)
//
// Channel Sizing Guidelines:
//   - msg_chan: Size based on expected message volume (typically 10-100)
//   - disconnect: Usually unbuffered or size 1 for coordination
//
// Example Usage:
//
//	msgChan := make(chan shared.Msg, 50)
//	disconnectChan := make(chan bool, 1)
//	handler := shared.NewBaseRobotHandler(robot, msgChan, disconnectChan)
func NewBaseRobotHandler(robot Robot, msg_chan chan Msg, disconnect chan bool) *BaseRobotHandler {
	if disconnect == nil {
		DebugPanic("Disconnect channel cannot be nil")
	}

	return &BaseRobotHandler{
		Robot:      robot,
		MsgChan:    msg_chan, // Example buffer size, adjust as needed
		disconnect: disconnect,
	}
}

// NewBaseRobotConnHandler creates a new BaseRobotConnHandler for connection management.
//
// This constructor initializes a connection handler that manages the lifecycle
// of robot connections. The handler coordinates between the robot's communication
// handler and the connection management system.
//
// Parameters:
//   - deviceId: Unique robot identifier
//   - ip: Robot's network address
//   - handler: Robot handler managing state and communication
//
// Returns:
//   - *BaseRobotConnHandler: Properly initialized connection handler
//
// The connection handler automatically extracts the disconnect channel from
// the provided handler to ensure proper coordination during shutdown.
//
// Example Usage:
//
//	connHandler := shared.NewBaseRobotConnHandler("robot_001", "192.168.1.100", robotHandler)
func NewBaseRobotConnHandler(deviceId string, ip string, handler RobotHandler) *BaseRobotConnHandler {
	return &BaseRobotConnHandler{
		DeviceID:       deviceId,
		IP:             ip,
		Handler:        handler,
		DisconnectChan: handler.GetDisconnectChannel(),
	}
}

// BaseRobot Method Implementations
//
// These methods implement the Robot interface for BaseRobot.

// ToJSON serializes the robot state to a JSON string for API responses.
//
// This method converts the robot's current state into a JSON representation
// suitable for transmission over network APIs or storage. The AuthToken field
// is automatically excluded from serialization for security.
//
// Returns:
//   - string: JSON representation of robot state, or "{}" if serialization fails
//
// Example Output:
//
//	{
//	  "device_id": "sensor_001",
//	  "ip": "192.168.1.100",
//	  "robot_type": "proximity_sensor",
//	  "status": "online",
//	  "battery": 85,
//	  "last_seen": 1672531200
//	}
//
// Thread Safety:
// This method is safe to call concurrently as it only reads robot state.
func (br *BaseRobot) ToJSON() string {
	data, err := json.Marshal(br)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// GetBaseRobot returns a copy of the embedded BaseRobot struct.
//
// This method provides access to the base robot data for functions that need
// to work with the common robot fields. It returns a copy to prevent
// external modification of the robot's internal state.
//
// Returns:
//   - BaseRobot: Copy of the robot's base structure
//
// Usage:
// This method is primarily used by the Robot interface implementation
// and for accessing common fields in generic robot handling code.
func (br *BaseRobot) GetBaseRobot() BaseRobot {
	return *br
}

// GetDeviceID returns the robot's unique device identifier.
//
// This method provides access to the robot's persistent identifier used
// for authentication, tracking, and robot management operations.
//
// Returns:
//   - string: The robot's unique device identifier
//
// Thread Safety:
// This method is safe to call concurrently as it only reads robot state.
func (br *BaseRobot) GetDeviceID() string {
	return br.DeviceID
}

// GetIP returns the robot's current IP address.
//
// This method provides access to the robot's network address used for
// communication. Note that IP addresses may change during a robot's
// lifetime due to network configuration changes.
//
// Returns:
//   - string: The robot's current IP address
//
// Thread Safety:
// This method is safe to call concurrently as it only reads robot state.
func (br *BaseRobot) GetIP() string {
	return br.IP
}

// IsOnline checks if the robot is currently connected and responsive.
//
// This method determines robot connectivity by checking the status field
// against known online status values. Different robot types may use
// different status terminology, so this method normalizes the check.
//
// Recognized Online Status Values:
//   - "online"
//   - "connected"
//   - "active"
//
// Returns:
//   - bool: true if robot is considered online, false otherwise
//
// Usage:
// This method is commonly used for:
// - Health monitoring and dashboards
// - Filtering available robots for commands
// - Connection state validation
//
// Thread Safety:
// This method is safe to call concurrently as it only reads robot state.
func (br *BaseRobot) IsOnline() bool {
	return br.Status == "online" || br.Status == "connected" || br.Status == "active"
}

// SetLastSeen updates the robot's last activity timestamp.
//
// This method is called whenever the robot communicates with the server
// to maintain an accurate record of robot connectivity and activity.
// The timestamp is used for health monitoring and connection timeout detection.
//
// Parameters:
//   - timestamp: Unix timestamp of the activity (typically time.Now().Unix())
//
// Usage:
// This method should be called:
// - When messages are received from the robot
// - During periodic heartbeat updates
// - When commands are successfully sent to the robot
//
// Thread Safety:
// This method modifies robot state and should be called with appropriate
// synchronization if the robot is accessed from multiple goroutines.
func (br *BaseRobot) SetLastSeen(timestamp int64) {
	br.LastSeen = timestamp
}

// String returns a human-readable representation of the robot for logging and debugging.
//
// This method provides a formatted string containing the robot's key identifying
// information and current state. It's particularly useful for log messages,
// debugging output, and diagnostic information.
//
// Returns:
//   - string: Formatted robot description with key fields
//
// Example Output:
//
//	"Robot(DeviceID: sensor_001, RobotType: proximity_sensor, IP: 192.168.1.100, Status: online, Battery: 85%, LastSeen: 1672531200)"
//
// Thread Safety:
// This method is safe to call concurrently as it only reads robot state.
func (br *BaseRobot) String() string {
	return fmt.Sprintf("Robot(DeviceID: %s, RobotType: %s, IP: %s, Status: %s, Battery: %d%%, LastSeen: %d)",
		br.DeviceID, br.RobotType, br.IP, br.Status, br.Battery, br.LastSeen)
}

// BaseRobotHandler Method Implementations
//
// These methods implement the RobotHandler interface for BaseRobotHandler.

// GetRobot returns the robot instance managed by this handler.
//
// This method provides access to the robot state for status queries,
// updates, and other robot-specific operations.
//
// Returns:
//   - Robot: The robot instance implementing the Robot interface
//
// Thread Safety:
// This method is safe to call concurrently as it returns a reference
// to the robot instance.
func (br *BaseRobotHandler) GetRobot() Robot {
	return br.Robot
}

// SendMsg queues a message for processing by the robot.
//
// This is a basic implementation that validates the message channel is initialized
// but does not implement actual message processing. Specific robot types should
// override this method to provide meaningful message handling.
//
// Parameters:
//   - msg: Message to send to the robot implementing the Msg interface
//
// Returns:
//   - error: ErrMsgChannelUninitialized if channel is nil, ErrMsgUnknownType otherwise
//
// Override Required:
// Robot-specific handlers should override this method to implement:
// - Actual message queuing to the robot's message channel
// - Message type validation and routing
// - Appropriate error handling for the robot type
//
// Example Override:
//
//	func (rh *SpecificRobotHandler) SendMsg(msg shared.Msg) error {
//	    if rh.MsgChan == nil {
//	        return shared.ErrMsgChannelUninitialized
//	    }
//	    select {
//	    case rh.MsgChan <- msg:
//	        return nil
//	    default:
//	        return errors.New("message queue full")
//	    }
//	}
func (br *BaseRobotHandler) SendMsg(msg Msg) error {
	if br.MsgChan == nil {
		return ErrMsgChannelUninitialized
	}
	<-br.MsgChan
	return ErrMsgUnknownType
}

// GetDeviceID returns the device ID of the robot managed by this handler.
//
// This is a convenience method that delegates to the underlying robot's
// GetDeviceID method, providing consistent access to the robot's identifier
// through the handler interface.
//
// Returns:
//   - string: The robot's unique device identifier
//
// Thread Safety:
// This method is safe to call concurrently.
func (br *BaseRobotHandler) GetDeviceID() string {
	return br.Robot.GetDeviceID()
}

// GetIP returns the IP address of the robot managed by this handler.
//
// This is a convenience method that delegates to the underlying robot's
// GetIP method, providing consistent access to the robot's network address
// through the handler interface.
//
// Returns:
//   - string: The robot's current IP address
//
// Thread Safety:
// This method is safe to call concurrently.
func (br *BaseRobotHandler) GetIP() string {
	return br.Robot.GetIP()
}

// GetDisconnectChannel returns the channel used for coordinating robot disconnection.
//
// This channel is used by the robot management system to signal when a robot
// should be disconnected and cleaned up. Components can listen on this channel
// to coordinate graceful shutdown.
//
// Returns:
//   - chan bool: Channel for disconnect coordination (never nil for valid handlers)
//
// Usage Patterns:
//
//	// Listen for disconnect signal
//	select {
//	case <-handler.GetDisconnectChannel():
//	    // Robot disconnected, clean up resources
//	case msg := <-msgChannel:
//	    // Process message
//	}
//
// Thread Safety:
// This method is safe to call concurrently.
func (br *BaseRobotHandler) GetDisconnectChannel() chan bool {
	return br.disconnect
}

// QuickAction performs a simple robot action for testing or health checking.
//
// This is a placeholder implementation for robot-specific quick actions.
// Different robot types can override this method to implement appropriate
// quick actions such as:
// - Status ping or health check
// - Battery level query
// - Simple movement or sensor reading
// - LED blink or audio beep
//
// The base implementation is a no-op that can be safely called but performs
// no actual action. Robot-specific implementations should provide meaningful
// functionality for their robot type.
//
// Example Overrides:
//
//	// Proximity sensor: take a distance reading
//	// Door opener: check door status
//	// Trash can: report fill level
//
// Thread Safety:
// Implementations should be thread-safe and non-blocking.
func (br *BaseRobotHandler) QuickAction() {
	// Base implementation: no-op
	// Robot-specific handlers should override this method
}

// BaseRobotConnHandler Method Implementations
//
// These methods implement the RobotConnHandler interface for BaseRobotConnHandler.

// Start begins the connection handling routine for the robot.
//
// This is a placeholder implementation that should be overridden by specific
// robot types to implement their communication protocols. The actual implementation
// should establish and maintain communication with the robot hardware.
//
// Typical implementations should:
// - Establish network connection to the robot
// - Start message processing loops
// - Handle protocol-specific communication
// - Monitor connection health
// - Process incoming sensor data or commands
//
// Returns:
//   - error: nil for success, specific error for connection failures
//
// Override Required:
// Robot-specific connection handlers must override this method to implement
// actual communication protocols for their robot type.
//
// Example Structure:
//
//	func (rc *SpecificConnHandler) Start() error {
//	    // Establish connection
//	    conn, err := net.Dial("tcp", rc.IP+":8080")
//	    if err != nil {
//	        return err
//	    }
//	    defer conn.Close()
//
//	    // Start message processing loop
//	    for {
//	        select {
//	        case <-rc.GetDisconnectChannel():
//	            return nil
//	        case msg := <-rc.Handler.GetMsgChan():
//	            // Process message
//	        }
//	    }
//	}
//
// Thread Safety:
// This method is expected to be called from a dedicated goroutine and should
// handle concurrent access to shared resources appropriately.
func (brc *BaseRobotConnHandler) Start() error {
	// Base implementation: no-op
	// Robot-specific connection handlers should override this method
	return nil
}

// Stop terminates the connection and cleans up associated resources.
//
// This method handles the graceful shutdown of the robot connection by closing
// the disconnect channel and performing any necessary cleanup. It's called
// when the robot is being removed from the system or the server is shutting down.
//
// Cleanup Operations:
// - Closes the disconnect channel to signal shutdown to other components
// - Can be extended by robot-specific handlers for additional cleanup
//
// Returns:
//   - error: nil for successful cleanup, specific error for cleanup failures
//
// Robot-specific implementations should override this method to:
// - Close network connections
// - Stop background goroutines
// - Release hardware resources
// - Save persistent state if needed
//
// Example Override:
//
//	func (rc *SpecificConnHandler) Stop() error {
//	    // Close robot-specific connections
//	    if rc.conn != nil {
//	        rc.conn.Close()
//	    }
//	    // Call base implementation
//	    return rc.BaseRobotConnHandler.Stop()
//	}
//
// Thread Safety:
// This method may be called concurrently with other handler methods and
// should handle synchronization appropriately.
func (brc *BaseRobotConnHandler) Stop() error {
	SafeClose(brc.DisconnectChan)
	return nil
}

// GetHandler returns the robot handler managed by this connection handler.
//
// This method provides access to the robot handler instance, which manages
// the robot's state and message processing. The handler is used for sending
// messages to the robot and accessing robot information.
//
// Returns:
//   - RobotHandler: The handler instance for robot communication and state management
//
// Thread Safety:
// This method is safe to call concurrently as it returns a reference to
// the handler instance.
func (brc *BaseRobotConnHandler) GetHandler() RobotHandler {
	return brc.Handler
}

// GetDisconnectChannel returns the channel for coordinating connection shutdown.
//
// This method provides access to the disconnect channel used to signal when
// the connection should be terminated. The channel is monitored by connection
// handling routines to coordinate graceful shutdown.
//
// Returns:
//   - chan bool: Channel for disconnect coordination (should not be nil)
//
// Usage:
// Connection handling routines typically monitor this channel in select
// statements to respond to shutdown signals:
//
//	select {
//	case <-connHandler.GetDisconnectChannel():
//	    return // Graceful shutdown
//	case data := <-networkData:
//	    // Process network data
//	}
//
// Thread Safety:
// This method is safe to call concurrently.
func (brc *BaseRobotConnHandler) GetDisconnectChannel() chan bool {
	return brc.DisconnectChan
}
