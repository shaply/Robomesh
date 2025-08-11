package http_server

import "roboserver/shared/robot_manager"

// /robot/register
type RegisterRobotRequest struct {
	Robot  robot_manager.RegisteringRobot `json:"registering_robot"`
	Accept string                         `json:"accept"` // "yes" or "no"
}
