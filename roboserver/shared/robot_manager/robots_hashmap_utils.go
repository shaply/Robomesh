package robot_manager

import (
	"roboserver/shared"
)

func (h *RobotHandler) AddRobot(robot *shared.BaseRobot) int { // Make handler to do this
	if robot == nil {
		return 0
	} else if robot.ID != 0 {
		return robot.ID // If the robot already has an ID, return it
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	robot.ID = h.generateRobotID()
	h.robotsByID[robot.ID] = robot
	h.robotsByIP[robot.IP] = robot
	return robot.ID
}

func (h *RobotHandler) RemoveRobot(id int) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	robot, exists := h.robotsByID[id]
	if !exists {
		return nil // or an error if you prefer
	}

	delete(h.robotsByID, id)
	delete(h.robotsByIP, robot.IP)
	return nil
}

func (h *RobotHandler) UpdateRobotStatus(id int, status string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	robot, exists := h.robotsByID[id]
	if !exists {
		return nil // or an error if you prefer
	}

	robot.Status = status
	return nil
}

func (h *RobotHandler) GetRobot(id int) (*shared.BaseRobot, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	robot, exists := h.robotsByID[id]
	return robot, exists
}

func (h *RobotHandler) GetRobotByIP(ip string) (*shared.BaseRobot, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	robot, exists := h.robotsByIP[ip]
	return robot, exists
}

func (h *RobotHandler) generateRobotID() int {
	if h.mu.TryLock() {
		defer h.mu.Unlock()
	} else {
		// If the lock is already held, we can safely assume idCounter is being used
	}
	id := h.idCounter + 1
	h.idCounter++
	return id
}
