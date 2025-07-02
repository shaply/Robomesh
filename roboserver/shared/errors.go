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
)
