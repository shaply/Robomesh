package tcp_server

import "net"

/*
`TRANSFER`: Stalls the TCP connection handler (so the TCP server stops reading), and gives the reading and writing functionality solely to the robot handling go routine.
When the robot is done with the transfer, it should send a message back to the TCP server to resume normal operation by writing to the reply channel.
*/
func handleTransfer(s *TCPServer, conn net.Conn, message string) {
	robot_handler, err := s.rm.GetHandler("", conn.RemoteAddr().String())
	if err != nil {
		conn.Write([]byte("ERROR NO_ROBOT_REGISTERED_WITH_IP\n"))
		return
	}

	replyChan := make(chan any, 1) // Create a reply channel for the message
	if err := robot_handler.SendMsg(NewTCPMessage(message, conn, replyChan)); err != nil {
		conn.Write([]byte("ERROR SENDING_MESSAGE_TO_ROBOT\n"))
		return
	} else {
		<-replyChan
	}
}
