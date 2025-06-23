package robot_manager

import (
	"roboserver/shared"
	"sync"
)

type RobotHandler struct {
	robotsByID map[int]*shared.Robot
	robotsByIP map[string]*shared.Robot
	idCounter  int
	mu         sync.RWMutex
}

func NewRobotHandler() *RobotHandler {
	return &RobotHandler{
		robotsByID: make(map[int]*shared.Robot),
		robotsByIP: make(map[string]*shared.Robot),
		idCounter:  0,
		mu:         sync.RWMutex{},
	}
}

func (h *RobotHandler) RegisterRobot(ip string, robot_type string, deviceID string) error {
	// Placeholder
	robot := NewRobot(0, "Robot", ip, robot_type, deviceID)
	if robot == nil {
		return nil // or an error if you prefer
	}

	robot.ID = h.AddRobot(robot)
	if robot.ID == 0 {
		return nil // or an error if you prefer
	}

	return h.UpdateRobotStatus(robot.ID, "online")
}

func (h *RobotHandler) GetRobots() []*shared.Robot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	robots := make([]*shared.Robot, 0, len(h.robotsByID))
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
