// Package robots provides registration and initialization for all robot types.
//
// This package serves as the central registry for robot implementations in the
// Robomesh system. It imports all robot type packages to ensure their init()
// functions are called during server startup, registering their factory functions
// with the global robot factory.
//
// Robot Registration Process:
// 1. Each robot type package defines its RobotType constant
// 2. The init() function calls shared.AddRobotType() to register the type
// 3. This package imports all robot types using blank imports
// 4. The main server imports this package to trigger registration
//
// Adding New Robot Types:
// 1. Create a new package under robots/ (e.g., robots/new_robot_type/)
// 2. Implement the required interfaces (Robot, RobotHandler, RobotConnHandler)
// 3. Add init() function to register the type with shared.AddRobotType()
// 4. Add blank import to this file: _ "roboserver/robots/new_robot_type"
//
// Currently Registered Robot Types:
// - proximity_sensor: Distance sensing and obstacle detection robots
package robots

import (
	_ "roboserver/robots/example_robot"
	_ "roboserver/robots/proximity_sensor" // Register proximity sensor robot type
)
