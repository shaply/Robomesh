package shared

import "errors"

// Robot Management Errors
var ErrRobotNotFound = errors.New("robot not found")
var ErrRobotBlacklisted = errors.New("robot is blacklisted")

// Authentication Errors
var ErrUnauthorized = errors.New("unauthorized access")
var ErrInvalidSignature = errors.New("invalid signature")
var ErrTokenExpired = errors.New("token expired")

// General Errors
var ErrInvalidInput = errors.New("invalid input provided")
var ErrHandlerNotFound = errors.New("handler script not found for device type")
