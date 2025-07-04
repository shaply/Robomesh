// Package shared provides utility functions for the Robomesh server.
//
// This file contains essential utility functions for network discovery, robot type
// registration, and safe resource cleanup. These utilities are used throughout
// the server for common operations that need to be handled consistently.
package shared

import (
	"net"
	"reflect"
	"sync"
)

// GetLocalIPs discovers and returns all local IPv4 addresses of the server.
//
// This function scans all network interfaces on the system and returns only
// active IPv4 addresses that can be used for robot communication. It filters
// out loopback addresses, IPv6 addresses, and interfaces that are down.
//
// The returned IP addresses can be used to:
// - Display available server endpoints to users
// - Configure robot network settings
// - Validate incoming connection sources
//
// Returns:
//   - []string: List of local IPv4 addresses in string format
//
// Example Usage:
//
//	ips := shared.GetLocalIPs()
//	for _, ip := range ips {
//	    fmt.Printf("Server available at: %s\n", ip)
//	}
func GetLocalIPs() []string {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		// Skip loopback and interfaces that are down
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip IPv6 and loopback addresses
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			ips = append(ips, ip.String())
		}
	}

	return ips
}

// AddRobotType registers a new robot type with its corresponding factory function.
//
// This function implements the factory pattern for robot creation by associating
// robot types with their constructor functions. Robot packages typically call this
// during initialization to register themselves with the system.
//
// The factory function will be used by RobotManager to create appropriate
// connection handlers when robots of this type connect to the server.
//
// Parameters:
//   - robotType: Unique identifier for the robot category
//   - newFunc: Factory function that creates connection handlers for this robot type
//
// Panics:
//   - If robotType is already registered (prevents accidental overwrites)
//   - If newFunc is nil (invalid factory function)
//
// Example Usage:
//
//	func init() {
//	    shared.AddRobotType("proximity_sensor", NewProximitySensorConnHandler)
//	}
//
// Thread Safety:
// This function is not thread-safe and should only be called during package
// initialization (in init() functions) before the server starts.
func AddRobotType(robotType RobotType, newFunc NewRobotConnHandlerFunc) {
	if _, exists := ROBOT_FACTORY[robotType]; exists {
		DebugPanic("Robot type already exists: " + string(robotType))
	}
	if newFunc == nil {
		DebugPanic("NewRobotConnHandlerFunc cannot be nil for robot type: " + string(robotType))
	}
	ROBOT_FACTORY[robotType] = newFunc
}

// channelCloseMutex protects against concurrent channel close operations.
// This prevents race conditions when multiple goroutines attempt to close
// the same channel simultaneously.
var channelCloseMutex sync.Mutex

// SafeClose safely closes various types of resources without panicking.
//
// This function provides a unified interface for closing different resource types:
// - Objects with Close() method (files, connections, etc.)
// - Channels (using reflection for type safety)
// - nil values (ignored safely)
//
// The function automatically detects the resource type and uses the appropriate
// closing mechanism. For channels, it uses SafeCloseChannel to prevent panics
// from attempting to close already-closed channels.
//
// Parameters:
//   - closer: Resource to close (can be nil, channel, or object with Close() method)
//
// Error Handling:
// Errors from Close() methods are logged but do not cause panics.
// Invalid resource types are handled gracefully.
//
// Example Usage:
//
//	defer shared.SafeClose(conn)        // TCP connection
//	defer shared.SafeClose(file)        // File handle
//	defer shared.SafeClose(msgChan)     // Channel
//	defer shared.SafeClose(nil)         // Safe, does nothing
//
// Thread Safety:
// This function is thread-safe for all supported resource types.
func SafeClose(closer interface{}) {
	if closer == nil {
		return
	}

	// Handle types with Close() method
	if c, ok := closer.(interface{ Close() error }); ok {
		if err := c.Close(); err != nil {
			DebugPrint("Error closing resource: %v", err)
		}
		return
	}

	// Handle channels using reflection
	SafeCloseChannel(closer)
}

// SafeCloseChannel safely closes a channel without panicking on already-closed channels.
//
// This function uses reflection to safely close channels of any type while preventing
// the common panic that occurs when attempting to close an already-closed channel.
// It includes mutex protection to handle concurrent close attempts.
//
// The function performs these safety checks:
// 1. Validates the input is actually a channel
// 2. Uses mutex to prevent concurrent close operations
// 3. Checks if channel is already closed before attempting to close
//
// Parameters:
//   - ch: Channel to close (must be a channel type)
//
// Behavior:
//   - nil channels are ignored safely
//   - Non-channel types are logged and ignored
//   - Already-closed channels are detected and ignored
//   - Concurrent close attempts are serialized with mutex
//
// Example Usage:
//
//	msgChan := make(chan string, 10)
//	defer shared.SafeCloseChannel(msgChan)
//
// Thread Safety:
// This function is thread-safe and can be called concurrently from multiple goroutines.
func SafeCloseChannel(ch interface{}) {
	if ch == nil {
		return
	}

	val := reflect.ValueOf(ch)
	if val.Kind() != reflect.Chan {
		DebugPrint("SafeCloseChannel: not a channel, type: %T", ch)
		return
	}

	channelCloseMutex.Lock()
	defer channelCloseMutex.Unlock()

	// Check if channel is closed by attempting a non-blocking receive
	if !isChannelClosed(val) {
		val.Close()
	}
}

// isChannelClosed checks if a channel is closed using non-blocking reflection.
//
// This function safely determines if a channel is closed without blocking or
// consuming values from the channel. It uses Go's reflection select mechanism
// to perform a non-blocking receive operation.
//
// The detection logic:
// - Sets up a select with the channel and a default case
// - If the channel case is chosen and ok=false, the channel is closed
// - If the default case is chosen, the channel is open but not ready
// - If the channel case is chosen and ok=true, the channel is open with data
//
// Parameters:
//   - ch: reflect.Value representing a channel to check
//
// Returns:
//   - bool: true if channel is closed, false if open
//
// Note: This function assumes ch.Kind() == reflect.Chan and should only be
// called after validating the channel type.
//
// Internal Use:
// This is a helper function for SafeCloseChannel and is not intended for
// direct external use.
func isChannelClosed(ch reflect.Value) bool {
	if ch.Kind() != reflect.Chan {
		return true
	}

	// Try non-blocking receive
	chosen, _, ok := reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: ch},
		{Dir: reflect.SelectDefault},
	})

	// If chosen == 0, we received from the channel
	// If chosen == 1, it was the default case (channel not ready)
	// If ok == false and chosen == 0, channel is closed
	return chosen == 0 && !ok
}
