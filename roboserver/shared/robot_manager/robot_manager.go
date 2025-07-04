// Package robot_manager provides centralized management of robot connections and state.
// It maintains dual-indexed maps for efficient robot lookup by both device ID and IP address,
// with thread-safe operations for concurrent access from multiple server components.
package robot_manager

import (
	"context"
	"roboserver/shared"
	"sync"
)

// RobotManager is the central coordinator for all robot connections in the system.
// It provides thread-safe operations for registering, managing, and communicating with robots.
//
// The manager maintains two synchronized maps for efficient lookups:
// - robotsByID: Quick access by unique device identifier
// - robotsByIP: Quick access by network address
//
// Thread Safety: All public methods are thread-safe using RWMutex for optimal concurrent access.
// Lifecycle: Robots are automatically cleaned up when disconnected or when main context is cancelled.
type RobotManager struct {
	robotsByID   map[string]shared.RobotHandler // Primary index: device ID -> robot handler
	robotsByIP   map[string]shared.RobotHandler // Secondary index: IP address -> robot handler
	mu           sync.RWMutex                   // Protects concurrent access to maps
	main_context context.Context                // Server-wide context for graceful shutdown coordination
}

// NewRobotManager creates a new RobotManager instance with the provided context.
//
// Parameters:
//   - main_context: The server's main context used for coordinating graceful shutdowns.
//     When this context is cancelled, all managed robots will be disconnected.
//
// Returns:
//   - *RobotManager: A new manager instance ready to handle robot registrations.
//
// Example:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	manager := NewRobotManager(ctx)
//	defer cancel() // This will trigger cleanup of all robots
func NewRobotManager(main_context context.Context) *RobotManager {
	return &RobotManager{
		robotsByID:   make(map[string]shared.RobotHandler),
		robotsByIP:   make(map[string]shared.RobotHandler),
		main_context: main_context,
	}
}

// AddRobot adds a robot handler to the manager with conflict resolution.
//
// This method handles several scenarios:
// 1. New robot registration - adds to both ID and IP maps
// 2. IP change for existing robot - updates IP mapping for same device
// 3. IP conflict resolution - removes stale registrations and retries
//
// Parameters:
//   - deviceId: Unique device identifier (e.g., "trash_robot_001")
//   - ip: Robot's current IP address (e.g., "192.168.1.100")
//   - handler: Robot handler implementation for communication
//
// Returns:
//   - error: nil on success, or one of:
//   - shared.ErrRobotAlreadyExists: Robot with same ID and IP already registered
//   - shared.ErrRobotTransfer: Robot changed IP address (operation succeeded)
//
// Thread Safety: This method is thread-safe and handles concurrent access.
//
// Security Note: IP conflicts are resolved by removing the old registration.
// TODO: Implement authentication tokens to prevent malicious robot impersonation.
//
// Example:
//
//	err := manager.AddRobot("robot_001", "192.168.1.100", handler)
//	if err == shared.ErrRobotTransfer {
//	    log.Println("Robot successfully moved to new IP")
//	}
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

// RemoveRobot safely removes a robot from the manager by device ID, IP, or both.
//
// This method provides flexible robot removal with validation:
// - Single identifier: Remove by device ID OR IP address
// - Dual identifier: Remove only if both ID and IP match the same robot
// - Cleanup: Automatically closes robot's disconnect channel
//
// Parameters:
//   - deviceId: Device identifier (empty string to ignore)
//   - ip: IP address (empty string to ignore)
//
// Returns:
//   - error: nil on success, or one of:
//   - shared.ErrInvalidInput: Both parameters are empty
//   - shared.ErrRobotMismatch: ID and IP don't refer to the same robot
//   - shared.ErrRobotNotFound: No robot found with given identifier(s)
//
// Usage Examples:
//
//	manager.RemoveRobot("robot_001", "")           // Remove by device ID
//	manager.RemoveRobot("", "192.168.1.100")      // Remove by IP
//	manager.RemoveRobot("robot_001", "192.168.1.100") // Remove if both match
//
// Thread Safety: This method is thread-safe with proper locking.
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

// GetRobots returns a snapshot of all currently registered robots.
//
// Returns:
//   - []shared.Robot: Slice containing all robot instances (not handlers)
//
// The returned slice is a copy and safe to modify without affecting the manager.
// For real-time robot state, use SendMessage or GetRobot for individual robots.
//
// Thread Safety: This method is thread-safe using read locks for minimal contention.
//
// Example:
//
//	robots := manager.GetRobots()
//	fmt.Printf("Currently managing %d robots\n", len(robots))
//	for _, robot := range robots {
//	    fmt.Printf("Robot: %s (%s)\n", robot.GetDeviceID(), robot.GetIP())
//	}
func (rm *RobotManager) GetRobots() []shared.Robot {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	robots := make([]shared.Robot, 0, len(rm.robotsByID))
	for _, handler := range rm.robotsByID {
		robots = append(robots, handler.GetRobot())
	}
	return robots
}

// GetRobot retrieves a specific robot by device ID, IP address, or both.
//
// Lookup Strategy:
// - Both provided: Validates that ID and IP refer to the same robot
// - ID only: Direct lookup by device identifier
// - IP only: Direct lookup by network address
//
// Parameters:
//   - deviceId: Unique device identifier (empty string to ignore)
//   - ip: IP address (empty string to ignore)
//
// Returns:
//   - shared.Robot: The robot instance if found
//   - error: nil on success, or one of:
//   - shared.ErrInvalidInput: Both parameters are empty
//   - shared.ErrRobotMismatch: ID and IP don't refer to the same robot
//   - shared.ErrRobotNotFound: No robot found with given identifier(s)
//
// Thread Safety: Uses read locks for concurrent access without blocking other reads.
//
// Example:
//
//	robot, err := manager.GetRobot("robot_001", "")
//	if err == nil {
//	    fmt.Printf("Robot status: %s\n", robot.GetStatus())
//	}
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

// GetDeviceIDs returns a list of all registered device identifiers.
//
// Returns:
//   - []string: Slice of device IDs currently managed by this instance
//
// Useful for:
// - Administrative interfaces showing connected robots
// - Health checks and monitoring
// - Debugging connection issues
//
// The returned slice is a copy and safe to modify.
// Thread Safety: Uses read locks for safe concurrent access.
func (rm *RobotManager) GetDeviceIDs() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	deviceIDs := make([]string, 0, len(rm.robotsByID))
	for deviceID := range rm.robotsByID {
		deviceIDs = append(deviceIDs, deviceID)
	}
	return deviceIDs
}

// GetIPs returns a list of all IP addresses with registered robots.
//
// Returns:
//   - []string: Slice of IP addresses currently in use by robots
//
// Useful for:
// - Network monitoring and diagnostics
// - IP conflict detection
// - Security auditing
//
// The returned slice is a copy and safe to modify.
// Thread Safety: Uses read locks for safe concurrent access.
func (rm *RobotManager) GetIPs() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	ips := make([]string, 0, len(rm.robotsByIP))
	for ip := range rm.robotsByIP {
		ips = append(ips, ip)
	}
	return ips
}

// SendMessage sends a message to a specific robot identified by device ID, IP, or both.
//
// Message Delivery:
// - Queues message in robot's channel for asynchronous processing
// - Non-blocking operation (fails immediately if robot's queue is full)
// - Validates robot identity when both ID and IP are provided
//
// Parameters:
//   - deviceId: Target device identifier (empty string to ignore)
//   - ip: Target IP address (empty string to ignore)
//   - msg: Message to send (see shared.Msg for structure)
//
// Returns:
//   - error: nil on successful queuing, or one of:
//   - shared.ErrInvalidInput: Both identifiers are empty
//   - shared.ErrRobotMismatch: ID and IP don't refer to the same robot
//   - shared.ErrRobotNotFound: No robot found with given identifier(s)
//   - Queue full error: Robot's message queue is full
//
// Message Types: See shared.Msg documentation for supported message formats.
// Thread Safety: Uses read locks for safe concurrent access.
//
// Example:
//
//	msg := shared.Msg{Msg: "START_TASK", Source: "scheduler"}
//	err := manager.SendMessage("robot_001", "", msg)
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

// GetHandler retrieves the internal robot handler for advanced operations.
//
// Robot handlers provide lower-level access for:
// - Direct channel communication
// - Connection state management
// - Protocol-specific operations
//
// Parameters:
//   - deviceId: Target device identifier (empty string to ignore)
//   - ip: Target IP address (empty string to ignore)
//
// Returns:
//   - shared.RobotHandler: Handler interface for direct robot communication
//   - error: nil on success, or standard lookup errors
//
// Warning: Direct handler access bypasses some safety checks.
// Prefer SendMessage() for normal robot communication.
//
// Thread Safety: Uses read locks for safe concurrent access.
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

// GetHandlers returns all robot handlers for bulk operations.
//
// Returns:
//   - []shared.RobotHandler: Slice of all robot handlers currently managed
//
// Use cases:
// - Broadcasting messages to all robots
// - Bulk status collection
// - Administrative operations
//
// The returned slice is a copy and safe to modify.
// Thread Safety: Uses read locks for safe concurrent access.
//
// Example:
//
//	handlers := manager.GetHandlers()
//	for _, handler := range handlers {
//	    go broadcastShutdown(handler)
//	}
func (rm *RobotManager) GetHandlers() []shared.RobotHandler {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	handlers := make([]shared.RobotHandler, 0, len(rm.robotsByID))
	for _, handler := range rm.robotsByID {
		handlers = append(handlers, handler)
	}
	return handlers
}

// RegisterRobot is the primary entry point for robot registration and lifecycle management.
//
// This method handles the complete robot registration workflow:
// 1. Creates appropriate connection handler based on robot type
// 2. Adds robot to manager with conflict resolution
// 3. Starts robot communication goroutines
// 4. Sets up graceful cleanup on disconnection or server shutdown
//
// Parameters:
//   - deviceID: Unique robot identifier (e.g., "trash_collector_001")
//   - ip: Robot's network address (e.g., "192.168.1.100")
//   - robotType: Robot type from shared.ROBOT_FACTORY (e.g., "trash", "door")
//
// Returns:
//   - error: nil on success, or one of:
//   - shared.ErrNoRobotTypeConnHandler: Unknown robot type
//   - shared.ErrCreateConnHandler: Failed to create connection handler
//   - shared.ErrRobotAlreadyExists: Robot already registered
//   - shared.ErrNoDisconnectChannel: Handler missing disconnect channel
//
// Lifecycle Management:
// - Automatically starts robot communication goroutines
// - Monitors for disconnection or server shutdown
// - Cleans up resources when robot disconnects
// - Handles graceful shutdown when main context is cancelled
//
// Thread Safety: All operations are thread-safe and non-blocking.
//
// Example:
//
//	err := manager.RegisterRobot("trash_001", "192.168.1.100", "trash")
//	if err != nil {
//	    log.Printf("Failed to register robot: %v", err)
//	}
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
