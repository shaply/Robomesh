package robot_manager

import (
	"roboserver/shared"
)

func NewRobot(id int, name string, ip string, robot_type string, deviceID string) *shared.Robot {
	robot := &shared.Robot{
		ID:       id,
		Name:     name,
		IP:       ip,
		Type:     robot_type,
		Status:   "offline",
		DeviceID: deviceID,
	}

	return robot
}
