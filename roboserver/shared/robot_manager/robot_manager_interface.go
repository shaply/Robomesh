package robot_manager

import (
	"net"
	"roboserver/shared"
)

// RobotManager interface defines the contract for robot management operations.
// Implementations provide thread-safe operations for robot lifecycle management,
// dual-indexed lookups by device ID and IP address, and message routing.
type RobotManager interface {
	// RegisterRobot registers a new robot to the manager.
	// Returns nil on success, or one of:
	// - ErrRobotNotAccepted: Robot registration was not accepted by the server
	// - ErrNoRobotTypeConnHandler: No connection handler found for robot type
	// - ErrCreateConnHandler: Failed to create connection handler
	// - ErrRobotAlreadyExists: Robot with same ID and IP already registered
	// - ErrNoDisconnectChannel: Connection handler has no disconnect channel
	RegisterRobot(deviceID string, ip string, robotType shared.RobotType, conn net.Conn) error

	// AddRobot registers a new robot with conflict resolution.
	// Returns ErrRobotAlreadyExists if already registered, ErrRobotTransfer if IP changed.
	AddRobot(deviceId string, ip string, handler shared.RobotHandler) error

	// RemoveRobot unregisters a robot by device ID, IP, or both.
	// Returns ErrRobotNotFound if not found, ErrRobotMismatch if ID/IP don't match.
	RemoveRobot(deviceId string, ip string) error

	// GetRobots returns a snapshot of all currently registered robots.
	// Returns a copy safe to modify without affecting the manager.
	GetRobots() []shared.Robot

	// GetRobot retrieves a specific robot by device ID, IP, or both.
	// Returns ErrInvalidInput if both empty, ErrRobotMismatch if ID/IP don't match.
	GetRobot(deviceId string, ip string) (shared.Robot, error)

	// GetDeviceIDs returns all registered device identifiers.
	// Useful for administrative interfaces and monitoring.
	GetDeviceIDs() []string

	// GetIPs returns all IP addresses with registered robots.
	// Useful for network monitoring and conflict detection.
	GetIPs() []string

	// GetRegisteringRobots returns robots currently in registration process.
	// Helps prevent duplicate registrations and provides registration status.
	GetRegisteringRobots() []RegisteringRobot

	// SendMessage queues a message for asynchronous delivery to a robot.
	// Non-blocking operation that fails if robot's message queue is full.
	SendMessage(deviceId string, ip string, msg shared.Msg) error

	// GetHandler retrieves the internal robot handler for advanced operations.
	// Provides lower-level access for direct channel communication and state management.
	GetHandler(deviceId string, ip string) (shared.RobotHandler, error)

	// GetHandlers returns all robot handlers for bulk operations.
	// Useful for broadcasting messages and administrative operations.
	GetHandlers() []shared.RobotHandler
}
