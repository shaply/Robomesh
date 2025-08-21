package examplerobot

import "roboserver/shared"

const PROXIMITY_SENSOR_ROBOT_TYPE shared.RobotType = "example_robot_robot"

func init() {
	// Register the default robot type with its connection handler
	shared.AddRobotType(PROXIMITY_SENSOR_ROBOT_TYPE, NewRobotConnHandlerFunc)
}
