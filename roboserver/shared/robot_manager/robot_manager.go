package robot_manager

import (
	"context"
	"roboserver/shared"
	"sync"
)

type RobotManager struct {
	robotsByID   map[string]shared.RobotHandler // Access by device ID
	robotsByIP   map[string]shared.RobotHandler // Access by IP address
	mu           sync.RWMutex
	main_context context.Context // Assuming shared.Context is a type that holds the main context for the server
}

func NewRobotManager(main_context context.Context) *RobotManager {
	return &RobotManager{
		robotsByID:   make(map[string]shared.RobotHandler),
		robotsByIP:   make(map[string]shared.RobotHandler),
		main_context: main_context,
	}
}

/**
 * This method serves to add a robot to the backend maps.
 */
func (rm *RobotManager) AddRobot(deviceId string, ip string, handler shared.RobotHandler) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
retry:
	if _, exists := rm.robotsByIP[ip]; exists {
		if existingHandler := rm.robotsByIP[ip]; existingHandler.GetDeviceID() != deviceId {
			rm.mu.Unlock()
			rm.RemoveRobot("", ip)
			rm.mu.Lock()
			goto retry
		} else {
			return shared.ErrRobotAlreadyExists
		}
	}

	// TODO: Fix this with authentication token, this is a weak point because a malicious user could register a robot with the same IP
	if _, exists := rm.robotsByID[deviceId]; exists {
		rm.robotsByIP[ip] = rm.robotsByID[deviceId]
		delete(rm.robotsByIP, rm.robotsByID[deviceId].GetIP()) // Remove old IP mapping
		return shared.ErrRobotTransfer
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
			handler := rm.robotsByID[deviceId]
			shared.SafeClose(handler.GetDisconnectChannel())
			delete(rm.robotsByID, deviceId)
			delete(rm.robotsByIP, ip)
			return nil
		}
	}
	if deviceId != "" {
		if handler, exists := rm.robotsByID[deviceId]; exists {
			shared.SafeClose(handler.GetDisconnectChannel())
			delete(rm.robotsByID, deviceId)
			delete(rm.robotsByIP, handler.GetIP()) // Assuming GetRobot() returns a BaseRobot with IP
			return nil
		}
		return shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}
	if ip != "" {
		if handler, exists := rm.robotsByIP[ip]; exists {
			shared.SafeClose(handler.GetDisconnectChannel())
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

func (rm *RobotManager) GetRobot(deviceId string, ip string) (shared.Robot, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if deviceId != "" && ip != "" {
		if rm.robotsByID[deviceId] != rm.robotsByIP[ip] {
			return nil, shared.ErrRobotMismatch // Assuming this error is defined in shared package
		} else {
			return rm.robotsByID[deviceId].GetRobot(), nil
		}
	}
	if deviceId != "" {
		if handler, exists := rm.robotsByID[deviceId]; exists {
			return handler.GetRobot(), nil
		}
		return nil, shared.ErrRobotNotFound // Assuming this error is defined in shared package
	}

	if ip != "" {
		if handler, exists := rm.robotsByIP[ip]; exists {
			return handler.GetRobot(), nil
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

/**
 * This method will serve as the entry point for registering a robot.
 * In the future, it will handle the registration process, and already registered robots will
 * go through this method to AddRobot. Otherwise, this will hold the logic for sending the
 * registration request to the frontend.
 *
 * This method also serves to register the robot with the RobotManager system like starting
 * its Go routine.
 */
func (rm *RobotManager) RegisterRobot(deviceID string, ip string, robotType shared.RobotType) error {
	shared.DebugPrint("Registering robot: %s with device ID: %s", robotType, deviceID)
	connFunc, ok := shared.ROBOT_FACTORY[robotType]
	if !ok {
		shared.DebugPrint("No connection handler for robotype: %s", robotType)
		return shared.ErrNoRobotTypeConnHandler
	}

	connHandler, err := connFunc(deviceID, ip)
	if err != nil {
		return shared.ErrCreateConnHandler
	}
	err = rm.AddRobot(deviceID, ip, connHandler.GetHandler())
	if err != nil {
		return shared.ErrRobotAlreadyExists
	}

	disconnect := connHandler.GetDisconnectChannel()
	if disconnect == nil {
		rm.RemoveRobot(deviceID, ip)
		shared.DebugPanic("No disconnect channel for robot type %s", robotType)
		return shared.ErrNoDisconnectChannel
	}
	go func() {
		defer shared.SafeClose(disconnect)
		if err := connHandler.Start(); err != nil {
			shared.DebugPrint("Error starting connection handler for robot type %s: %v", robotType, err)
			return
		}
	}()
	go func() {
		select {
		case <-rm.main_context.Done():
			shared.SafeClose(disconnect)
		case <-disconnect:
		}
		shared.DebugPrint("Connection handler for robot %s disconnected", deviceID)
		connHandler.Stop()
		rm.RemoveRobot(deviceID, ip)
	}()

	return nil
}
