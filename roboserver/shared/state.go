// Package shared provides global state management for the Robomesh server.
//
// This file contains the robot factory registry that maps robot types to their
// corresponding constructor functions. This enables the factory pattern for
// dynamic robot creation based on type identification.
package shared

// ROBOT_FACTORY is the global registry mapping robot types to their factory functions.
//
// This map enables the factory pattern for robot creation by associating each
// robot type with its corresponding constructor function. When a robot connects,
// the RobotManager looks up the appropriate factory function based on the
// robot's declared type.
//
// Registration:
// Robot packages register themselves during initialization:
//
//	func init() {
//	    shared.AddRobotType("door_opener", NewDoorOpenerConnHandler)
//	}
//
// Usage:
// The RobotManager uses this map to create appropriate handlers:
//
//	factory, exists := ROBOT_FACTORY[robotType]
//	if exists {
//	    handler, err := factory(deviceID, ip)
//	}
//
// Thread Safety:
// This map should only be modified during package initialization (init functions)
// before the server starts accepting connections. No additional synchronization
// is needed if this convention is followed.
//
// Example Registered Types:
// - "base_robot": Generic robot with basic functionality
// - "proximity_sensor": Robot with distance sensing capabilities
// - "door_opener": Robot that can control door mechanisms
// - "trash_can": Smart waste management robot
var (
	ROBOT_FACTORY = map[RobotType]NewRobotConnHandlerFunc{
		// Robot types are registered here during package initialization
		// Example: DOOR_OPENER: NewDoorOpenerConnHandler,
	}
)
