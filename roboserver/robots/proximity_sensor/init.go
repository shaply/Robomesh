package proximity_sensor

import "roboserver/shared"

const PROXIMITY_SENSOR_ROBOT_TYPE shared.RobotType = "proximity_sensor_robot"

func init() {
	// Register the default robot type with its connection handler
	shared.AddRobotType(PROXIMITY_SENSOR_ROBOT_TYPE, NewRobotConnHandlerFunc)
}
