// Package shared defines custom error types for the Robomesh server.
//
// This file contains all application-specific errors used throughout the server
// for consistent error handling and reporting. Errors are categorized by
// functional area: robot management, authentication, communication, etc.
package shared

import "errors"

// Robot Management Errors
//
// These errors relate to robot registration, discovery, and lifecycle management.

// ErrRobotAlreadyExists indicates a robot with the same identifier is already registered.
// This typically occurs during registration when a device ID or IP is already in use.
var ErrRobotAlreadyExists = errors.New("robot already exists")

// ErrRobotNotFound indicates the requested robot could not be found in the system.
// This can occur when querying by device ID or IP address.
var ErrRobotNotFound = errors.New("robot not found")

// ErrIPAlreadyInUse indicates an IP address is already associated with another robot.
// This helps prevent IP conflicts in the robot management system.
var ErrIPAlreadyInUse = errors.New("IP address already in use")

// ErrRobotMismatch indicates inconsistency between device ID and IP address mappings.
// This suggests potential data corruption or authentication issues.
var ErrRobotMismatch = errors.New("robot mismatch between device ID and IP address")

// ErrRobotTransfer indicates a robot has moved to a new IP address.
// The system detected the same device ID connecting from a different IP.
var ErrRobotTransfer = errors.New("robot transfer detected, IP address already in use by another robot")

// Robot Type and Handler Errors
//
// These errors relate to robot type registration and handler creation.

// ErrInvalidRobotType indicates an unsupported or unrecognized robot type.
var ErrInvalidRobotType = errors.New("invalid robot type")

// ErrCreateConnHandler indicates failure to create a connection handler for a robot.
// This can occur due to resource constraints or configuration issues.
var ErrCreateConnHandler = errors.New("failed to create connection handler for robot")

// ErrNoRobotTypeConnHandler indicates no factory function is registered for the robot type.
// This typically means the robot package wasn't imported or properly initialized.
var ErrNoRobotTypeConnHandler = errors.New("no connection handler for the specified robot type")

// Communication Errors
//
// These errors relate to message passing and communication channels.

// ErrMsgChannelUninitialized indicates a message channel is not properly set up.
var ErrMsgChannelUninitialized = errors.New("message channel is not initialized")

// ErrMsgUnknownType indicates an unrecognized message type was received.
var ErrMsgUnknownType = errors.New("unknown message type received")

// ErrNoDisconnectChannel indicates no disconnect channel is available for coordination.
// This is critical for proper cleanup when robots disconnect.
var ErrNoDisconnectChannel = errors.New("no disconnect channel available for the robot")

// General Errors
//
// These errors apply to multiple functional areas.

// ErrUnauthorized indicates insufficient permissions for the requested operation.
var ErrUnauthorized = errors.New("unauthorized access")

// ErrInvalidCommand indicates an unrecognized or malformed command was received.
var ErrInvalidCommand = errors.New("invalid command")

// ErrInvalidInput indicates invalid parameters were provided to a function.
var ErrInvalidInput = errors.New("invalid input provided")

// Define custom errors for robot management
var ()
