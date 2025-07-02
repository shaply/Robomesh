package robot_manager

import (
	"roboserver/shared"
	"sync"
)

type RobotManager struct {
	robotsByID map[string]shared.RobotHandler // Access by device ID
	robotsByIP map[string]shared.RobotHandler // Access by IP address
	mu         sync.RWMutex
}

func NewRobotManager() *RobotManager {
	return &RobotManager{
		robotsByID: make(map[string]shared.RobotHandler),
		robotsByIP: make(map[string]shared.RobotHandler),
	}
}

func (rm *RobotManager) AddRobot(deviceId string, ip string, handler shared.RobotHandler) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.robotsByID[deviceId]; exists {
		return shared.ErrRobotAlreadyExists // Assuming this error is defined in shared package
	}

	if _, exists := rm.robotsByIP[ip]; exists {
		return shared.ErrIPAlreadyInUse // Assuming this error is defined in shared package
	}

	rm.robotsByID[deviceId] = handler
	rm.robotsByIP[ip] = handler

	return nil
}

/*
RemoveRobot removes a robot from the manager by its device ID or IP address.
Input empty string for deviceId or ip will remove the robot by the other identifier.
If both are provided, it will check if they match and remove the robot if they do.
*/
func (rm *RobotManager) RemoveRobot(deviceId string, ip string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if deviceId == "" && ip == "" {
		return shared.ErrInvalidInput // Assuming this error is defined in shared package
	}

	if deviceId != "" && ip != "" {
		if rm.robotsByID[deviceId] != rm.robotsByIP[ip] {
			return shared.ErrRobotMismatch // Assuming this error is defined in shared package
		} else {
			delete(rm.robotsByID, deviceId)
			delete(rm.robotsByIP, ip)
			return nil
		}
	}
	if deviceId != "" {
		if handler, exists := rm.robotsByID[deviceId]; exists {
			delete(rm.robotsByID, deviceId)
			delete(rm.robotsByIP, handler.GetIP()) // Assuming GetRobot() returns a BaseRobot with IP
			return nil
		}
		return shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}
	if ip != "" {
		if handler, exists := rm.robotsByIP[ip]; exists {
			delete(rm.robotsByIP, ip)
			delete(rm.robotsByID, handler.GetDeviceID()) // Assuming GetRobot() returns a BaseRobot with DeviceID
			return nil
		}
		return shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}
	return shared.ErrInvalidInput // Assuming this error is defined in shared package
}

func (rm *RobotManager) GetRobots() []shared.Robot {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	robots := make([]shared.Robot, 0, len(rm.robotsByID))
	for _, handler := range rm.robotsByID {
		robots = append(robots, handler.GetRobot())
	}
	return robots
}

func (rm *RobotManager) GetRobot(deviceId string, ip string) (shared.RobotHandler, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if deviceId != "" && ip != "" {
		if rm.robotsByID[deviceId] != rm.robotsByIP[ip] {
			return nil, shared.ErrRobotMismatch // Assuming this error is defined in shared package
		} else {
			return rm.robotsByID[deviceId], nil
		}
	}
	if deviceId != "" {
		if handler, exists := rm.robotsByID[deviceId]; exists {
			return handler, nil
		}
		return nil, shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}

	if ip != "" {
		if handler, exists := rm.robotsByIP[ip]; exists {
			return handler, nil
		}
		return nil, shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}

	return nil, shared.ErrInvalidInput // Assuming this error is defined in shared package
}

func (rm *RobotManager) GetDeviceIDs() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	deviceIDs := make([]string, 0, len(rm.robotsByID))
	for deviceID := range rm.robotsByID {
		deviceIDs = append(deviceIDs, deviceID)
	}
	return deviceIDs
}

func (rm *RobotManager) GetIPs() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	ips := make([]string, 0, len(rm.robotsByIP))
	for ip := range rm.robotsByIP {
		ips = append(ips, ip)
	}
	return ips
}

func (rm *RobotManager) SendMessage(deviceId string, ip string, msg shared.Msg) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if deviceId != "" && ip != "" {
		if rm.robotsByID[deviceId] != rm.robotsByIP[ip] {
			return shared.ErrRobotMismatch // Assuming this error is defined in shared package
		} else {
			return rm.robotsByID[deviceId].SendMsg(msg)
		}
	}
	if deviceId != "" {
		if handler, exists := rm.robotsByID[deviceId]; exists {
			return handler.SendMsg(msg)
		}
		return shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}

	if ip != "" {
		if handler, exists := rm.robotsByIP[ip]; exists {
			return handler.SendMsg(msg)
		}
		return shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}

	return shared.ErrInvalidInput // Assuming this error is defined in shared package
}

func (rm *RobotManager) GetHandler(deviceId string, ip string) (shared.RobotHandler, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if deviceId != "" && ip != "" {
		if rm.robotsByID[deviceId] != rm.robotsByIP[ip] {
			return nil, shared.ErrRobotMismatch // Assuming this error is defined in shared package
		} else {
			return rm.robotsByID[deviceId], nil
		}
	}
	if deviceId != "" {
		if handler, exists := rm.robotsByID[deviceId]; exists {
			return handler, nil
		}
		return nil, shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}

	if ip != "" {
		if handler, exists := rm.robotsByIP[ip]; exists {
			return handler, nil
		}
		return nil, shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}

	return nil, shared.ErrInvalidInput // Assuming this error is defined in shared package
}

func (rm *RobotManager) GetHandlers() []shared.RobotHandler {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	handlers := make([]shared.RobotHandler, 0, len(rm.robotsByID))
	for _, handler := range rm.robotsByID {
		handlers = append(handlers, handler)
	}
	return handlers
}
