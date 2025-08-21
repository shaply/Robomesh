package examplerobot

import "roboserver/shared"

func NewRobotHandler(robot *robot) *robothandler {
	return &robothandler{
		*shared.NewBaseRobotHandlerWithDisconnect(robot, make(chan shared.Msg, 1)),
	}
}

type robothandler struct {
	shared.BaseRobotHandler
}
