package robot_manager

import (
	"roboserver/shared"
	"sync"
)

type RobotHandler struct {
	robotsByID map[int]*shared.BaseRobot
	robotsByIP map[string]*shared.BaseRobot
	idCounter  int
	mu         sync.RWMutex
}

func NewRobotHandler() *RobotHandler {
	return &RobotHandler{
		robotsByID: make(map[int]*shared.BaseRobot),
		robotsByIP: make(map[string]*shared.BaseRobot),
		idCounter:  0,
		mu:         sync.RWMutex{},
	}
}

func (h *RobotHandler) RegisterRobot(ip string, robot_type string, deviceID string) error {
	// Placeholder
	robot := shared.NewBaseRobot("Robot", robot_type, ip, deviceID)
	if robot == nil {
		return nil // or an error if you prefer
	}

	robot.ID = h.AddRobot(robot)
	if robot.ID == 0 {
		return nil // or an error if you prefer
	}

	return h.UpdateRobotStatus(robot.ID, "online")
}

func (h *RobotHandler) GetRobots() []*shared.BaseRobot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	robots := make([]*shared.BaseRobot, 0, len(h.robotsByID))
	for _, robot := range h.robotsByID {
		robots = append(robots, robot)
	}
	return robots
}

func (h *RobotHandler) UnregisterRobot(id int) error {
	return h.RemoveRobot(id)
}

func (h *RobotHandler) UnregisterRobotByIP(ip string) error {
	robot, exists := h.GetRobotByIP(ip)
	if !exists {
		return nil // or an error if you prefer
	}
	return h.RemoveRobot(robot.ID)
}
