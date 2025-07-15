package tcp_server

import (
	"net"
	"roboserver/shared"
)

func handleRegister(s *TCPServer_t, conn net.Conn, args []string) {
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
	if err := s.rm.RegisterRobot(deviceID, conn.RemoteAddr().(*net.TCPAddr).IP.String(), robotType, conn); err != nil {
		switch err {
		case shared.ErrNoRobotTypeConnHandler:
			conn.Write([]byte("ERROR NO_ROBOT_TYPE_CONN_HANDLER\n"))
		case shared.ErrCreateConnHandler:
			conn.Write([]byte("ERROR CREATE_CONN_HANDLER\n"))
		case shared.ErrRobotAlreadyExists:
			conn.Write([]byte("ERROR ROBOT_ALREADY_EXISTS\n"))
		case shared.ErrNoDisconnectChannel:
			conn.Write([]byte("ERROR NO_DISCONNECT_CHANNEL\n"))
		default:
			conn.Write([]byte("ERROR UNKNOWN\n"))
		}
		return
	}

	shared.DebugPrint("Robot registered successfully: %s (%s)", robotType, deviceID)
	conn.Write([]byte("OK\n"))
}
