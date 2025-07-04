package proximity_sensor

import (
	"roboserver/shared"
	"time"
)

func NewRobotInit(deviceID string, ip string) *robot {
	// Create a new robot instance with the default BaseRobot
	return &robot{
		*shared.NewBaseRobot(deviceID, ip, PROXIMITY_SENSOR_ROBOT_TYPE, "online", 0, time.Now().Unix(), ""),
	}
}

type robot struct {
	shared.BaseRobot // Embed BaseRobot to inherit its fields and methods
}
