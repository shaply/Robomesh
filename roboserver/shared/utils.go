package shared

// AddRobotType registers a new robot type with its corresponding factory function.
//
// This function implements the factory pattern for robot creation by associating
// robot types with their constructor functions. Robot packages typically call this
// during initialization to register themselves with the system.
//
// The factory function will be used by RobotManager to create appropriate
// connection handlers when robots of this type connect to the server.
//
// Parameters:
//   - robotType: Unique identifier for the robot category
//   - newFunc: Factory function that creates connection handlers for this robot type
//
// Panics:
//   - If robotType is already registered (prevents accidental overwrites)
//   - If newFunc is nil (invalid factory function)
//
// Example Usage:
//
//	func init() {
//	    shared.AddRobotType("proximity_sensor", NewProximitySensorConnHandler)
//	}
//
// Thread Safety:
// This function is not thread-safe and should only be called during package
// initialization (in init() functions) before the server starts.
func AddRobotType(robotType RobotType, newFunc NewRobotConnHandlerFunc) {
	if _, exists := ROBOT_FACTORY[robotType]; exists {
		DebugPanic("Robot type already exists: " + string(robotType))
	}
	if newFunc == nil {
		DebugPanic("NewRobotConnHandlerFunc cannot be nil for robot type: " + string(robotType))
	}
	ROBOT_FACTORY[robotType] = newFunc
}
