package tcp_server

import (
	"net"
)

func handleDefault(s *TCPServer_t, conn net.Conn, message string) {
	robot_handler := s.validateRobot(conn.RemoteAddr().(*net.TCPAddr).IP.String())
	if robot_handler == nil {
		conn.Write([]byte("ERROR NO_ROBOT_REGISTERED_WITH_IP\n"))
		return
	}

	robot_handler.SendMsg(NewTCPMessage(message, conn, nil))
}
