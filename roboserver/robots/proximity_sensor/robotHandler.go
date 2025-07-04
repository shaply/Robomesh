package proximity_sensor

import "roboserver/shared"

func NewRobotHandler(robot *robot) *robothandler {
	return &robothandler{
		*shared.NewBaseRobotHandler(robot, make(chan shared.Msg, 1), make(chan bool)),
	}
}

type robothandler struct {
	shared.BaseRobotHandler
}
