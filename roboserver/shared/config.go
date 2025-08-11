// Package shared provides configuration management for the Robomesh server.
//
// This file handles server configuration through environment variables,
// particularly debug mode settings that control logging verbosity and
// development features throughout the application.
package shared

import (
	"os"
	"time"
)

// DEBUG_MODE controls debug logging and development features throughout the server.
//
// When true, enables:
// - Detailed debug output with file/line information
// - Additional runtime checks and validations
// - Verbose error reporting
// - Development-specific behavior
//
// This variable is set during server initialization based on the DEBUG
// environment variable and should not be modified at runtime.
var (
	DEBUG_MODE = false
)

const (
	MONGODB_MIN_POOL_SIZE = 2
	MONGODB_MAX_POOL_SIZE = 10

	REGISTERING_WAIT_TIMEOUT = 30 * time.Minute

	EVENT_BUS_BUFFER_SIZE = 1000 // Buffer size for event bus to handle high-frequency events
)

// InitConfig initializes server configuration from environment variables.
//
// This function should be called once during server startup to load
// configuration settings from the environment. Currently handles:
//
// Environment Variables:
//   - DEBUG: Set to "true" to enable debug mode and verbose logging
//
// Example Usage:
//
//	func main() {
//	    shared.InitConfig()  // Load config before starting servers
//	    // ... start servers
//	}
//
// Future Expansion:
// This function can be extended to handle additional configuration
// options like port numbers, authentication settings, and feature flags.
func InitConfig() {
	DEBUG_MODE = os.Getenv("DEBUG") == "true"
}
