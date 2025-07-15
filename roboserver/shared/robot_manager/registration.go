package robot_manager

import (
	"fmt"
	"net"
	"roboserver/shared"
	"roboserver/shared/event_bus"
	"time"
)

type RegisteringRobot struct {
	DeviceID  string           `json:"device_id"`    // Unique identifier for robot authentication and tracking
	IP        string           `json:"ip,omitempty"` // Current IP address for network communication
	RobotType shared.RobotType `json:"robot_type"`   // Robot category determining capabilities and handlers
}

// HandleRegister is a method to handle the registration of a robot.
// It publishes an event to the event bus with the registration details.
func (r RegisteringRobot) HandleRegister(eb event_bus.EventBus, acceptance bool) {
	eb.Publish(event_bus.NewDefaultEvent(
		fmt.Sprintf("register.%s%s%s", r.DeviceID, r.IP, r.RobotType),
		acceptance,
	))
}

type EventRegisteringRobot struct {
	Type string           `json:"type"` // Event type identifier
	Data RegisteringRobot `json:"data"` // Data associated with the event
}

func NewEventRegisteringRobot(reg *RegisteringRobot) *EventRegisteringRobot {
	return &EventRegisteringRobot{
		Type: "robot_manager.registering_robot",
		Data: *reg,
	}
}

// This method is designed to handle the stage in the registration of a robot
// where the robot needs to be accepted by a server.
// Returns whether the robot was accepted or not.
func (rm *RobotManager_t) handleRegisteringRobotEvent(deviceID string, ip string, robotType shared.RobotType) bool {
	// Create a new RegisteringRobot instance
	regRobot := RegisteringRobot{
		DeviceID:  deviceID,
		IP:        ip,
		RobotType: robotType,
	}

	eventString := fmt.Sprintf("register.%s%s%s", deviceID, ip, robotType)
	channel := make(chan bool)
	sub := rm.eventBus.Subscribe(eventString, nil, func(event event_bus.Event) {
		b, ok := event.GetData().(bool)
		if !ok {
			b = false
		}
		select {
		case channel <- b:
		default:
		}
	})

	// Add to the set of registering robots
	rm.registeringRobots.Add(regRobot)
	rm.eventBus.Publish(NewEventRegisteringRobot(&regRobot))

	accepted := false
	select {
	case accepted = <-channel:
	case <-rm.main_context.Done():
		accepted = false
	case <-time.After(shared.REGISTERING_WAIT_TIMEOUT):
		accepted = false
	}

	close(channel)
	rm.eventBus.Unsubscribe(eventString, sub)
	rm.registeringRobots.Remove(regRobot)

	return accepted
}

func (e *EventRegisteringRobot) GetType() string {
	return e.Type
}

func (e *EventRegisteringRobot) GetData() interface{} {
	return e.Data
}

// RegisterRobot is the primary entry point for robot registration and lifecycle management.
//
// This method handles the complete robot registration workflow:
// 1. Creates appropriate connection handler based on robot type
// 2. Adds robot to manager with conflict resolution
// 3. Starts robot communication goroutines
// 4. Sets up graceful cleanup on disconnection or server shutdown
//
// Parameters:
//   - deviceID: Unique robot identifier (e.g., "trash_collector_001")
//   - ip: Robot's network address (e.g., "192.168.1.100")
//   - robotType: Robot type from shared.ROBOT_FACTORY (e.g., "trash", "door")
//
// Returns:
//   - error: nil on success, or one of:
//   - shared.ErrNoRobotTypeConnHandler: Unknown robot type
//   - shared.ErrCreateConnHandler: Failed to create connection handler
//   - shared.ErrRobotAlreadyExists: Robot already registered
//   - shared.ErrNoDisconnectChannel: Handler missing disconnect channel
//
// Lifecycle Management:
// - Automatically starts robot communication goroutines
// - Monitors for disconnection or server shutdown
// - Cleans up resources when robot disconnects
// - Handles graceful shutdown when main context is cancelled
//
// Thread Safety: All operations are thread-safe and non-blocking.
//
// Example:
//
//	err := manager.RegisterRobot("trash_001", "192.168.1.100", "trash")
//	if err != nil {
//	    log.Printf("Failed to register robot: %v", err)
//	}
func (rm *RobotManager_t) RegisterRobot(deviceID string, ip string, robotType shared.RobotType, conn net.Conn) error {
	shared.DebugPrint("Registering robot: %s with device ID: %s", robotType, deviceID)

	conn.Write([]byte("REGISTERING\n"))
	accepted := rm.handleRegisteringRobotEvent(deviceID, ip, robotType)
	if !accepted {
		shared.DebugPrint("Robot registration not accepted: %s", deviceID)
		return shared.ErrRobotNotAccepted
	}

	connFunc, ok := shared.ROBOT_FACTORY[robotType]
	if !ok {
		shared.DebugPrint("No connection handler for robotype: %s", robotType)
		return shared.ErrNoRobotTypeConnHandler
	}

	connHandler, err := connFunc(deviceID, ip)
	if err != nil {
		return shared.ErrCreateConnHandler
	}
	err = rm.AddRobot(deviceID, ip, connHandler.GetHandler())
	if err != nil {
		return shared.ErrRobotAlreadyExists
	}

	disconnect := connHandler.GetDisconnectChannel()
	if disconnect == nil {
		rm.RemoveRobot(deviceID, ip)
		shared.DebugPanic("No disconnect channel for robot type %s", robotType)
		return shared.ErrNoDisconnectChannel
	}
	go func() {
		defer shared.SafeClose(disconnect)
		if err := connHandler.Start(); err != nil {
			shared.DebugPrint("Error starting connection handler for robot type %s: %v", robotType, err)
			return
		}
	}()
	go func() {
		select {
		case <-rm.main_context.Done():
			shared.SafeClose(disconnect)
		case <-disconnect:
		}
		shared.DebugPrint("Connection handler for robot %s disconnected", deviceID)
		connHandler.Stop()
		rm.RemoveRobot(deviceID, ip)
	}()

	return nil
}
