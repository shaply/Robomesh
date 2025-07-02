package shared

import (
	"net"
	"reflect"
	"sync"
)

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

func AddRobotType(robotType RobotType, newFunc NewRobotConnHandlerFunc) {
	if _, exists := ROBOT_FACTORY[robotType]; exists {
		DebugPanic("Robot type already exists: " + string(robotType))
	}
	if newFunc == nil {
		DebugPanic("NewRobotConnHandlerFunc cannot be nil for robot type: " + string(robotType))
	}
	ROBOT_FACTORY[robotType] = newFunc
}

var channelCloseMutex sync.Mutex

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
