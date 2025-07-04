package shared

import "errors"

// Define custom errors for robot management
var (
	ErrRobotAlreadyExists      = errors.New("robot already exists")
	ErrRobotNotFound           = errors.New("robot not found")
	ErrIPAlreadyInUse          = errors.New("IP address already in use")
	ErrInvalidRobotType        = errors.New("invalid robot type")
	ErrUnauthorized            = errors.New("unauthorized access")
	ErrInvalidCommand          = errors.New("invalid command")
	ErrInvalidInput            = errors.New("invalid input provided")
	ErrRobotMismatch           = errors.New("robot mismatch between device ID and IP address")
	ErrMsgChannelUninitialized = errors.New("message channel is not initialized")
	ErrMsgUnknownType          = errors.New("unknown message type received")
	ErrRobotTransfer           = errors.New("robot transfer detected, IP address already in use by another robot")
	ErrCreateConnHandler       = errors.New("failed to create connection handler for robot")
	ErrNoRobotTypeConnHandler  = errors.New("no connection handler for the specified robot type")
	ErrNoDisconnectChannel     = errors.New("no disconnect channel available for the robot")
)
