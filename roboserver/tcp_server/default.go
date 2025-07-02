package tcp_server

import (
	"net"
)

func handleDefault(s *TCPServer, conn net.Conn, message string) {
	robot_handler, err := s.rm.GetHandler("", conn.RemoteAddr().String())
	if err != nil {
		conn.Write([]byte("ERROR NO_ROBOT_REGISTERED_WITH_IP\n"))
		return
	}

	robot_handler.SendMsg(NewTCPMessage(message, conn, nil))
}
