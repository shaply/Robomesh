package tcp_server

import (
	"net"
	"roboserver/shared"
)

func handleRegister(s *TCPServer, conn net.Conn, args []string) {
	if len(args) < 3 {
		conn.Write([]byte("ERROR REGISTER\n"))
		shared.DebugPrint("tcp_server/register.go", 10, "Invalid REGISTER command format. Expected: REGISTER <robot_type> <device_id>")
		return
	}
	robotTypeStr := args[1]
	robotType := shared.RobotType(robotTypeStr)
	if robotType == "" {
		shared.DebugPrint("tcp_server/register.go", 15, "Invalid robot type: %s", robotTypeStr)
		conn.Write([]byte("ERROR INVALID_ROBOT_TYPE\n"))
		return
	}

	deviceID := args[2]
	shared.DebugPrint("Registering robot: %s with device ID: %s", robotType, deviceID)
	connFunc, ok := shared.ROBOT_FACTORY[robotType]
	if !ok {
		shared.DebugPrint("No connection handler for robotype: %s", robotType)
		conn.Write([]byte("ERROR NO_ROBOTYPE_CONN_HANDLER\n"))
		return
	}

	connHandler, err := connFunc(deviceID, conn.RemoteAddr().String())
	if err != nil {
		shared.DebugPrint("Error creating connection handler for robot type %s: %v", robotType, err)
		conn.Write([]byte("ERROR CREATE_CONN_HANDLER\n"))
		return
	}
	s.rm.AddRobot(deviceID, conn.RemoteAddr().String(), connHandler.GetHandler())
	disconnect := connHandler.GetDisconnectChannel()
	if disconnect == nil {
		conn.Write([]byte("ERROR NO_DISCONNECT_CHANNEL\n"))
		shared.DebugPanic("No disconnect channel for robot type %s", robotType)
		return
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
		case <-s.main_context.Done():
			shared.SafeClose(disconnect)
		case <-disconnect:
		}
		shared.DebugPrint("Connection handler for robot %s disconnected", deviceID)
		connHandler.Stop()
		s.rm.RemoveRobot(deviceID, conn.RemoteAddr().String())
	}()

	shared.DebugPrint("Robot registered successfully: %s (%s)", robotType, deviceID)
	conn.Write([]byte("OK\n"))
}
