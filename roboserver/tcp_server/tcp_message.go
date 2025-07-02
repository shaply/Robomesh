package tcp_server

import (
	"net"
	"roboserver/shared"
)

/*
A TCP message provides conn through GetConn() to write a reply. The source is always TCP_SERVER.
*/
type TCPMessage struct {
	shared.DefaultMsg
	conn net.Conn // The connection associated with this message, to write a reply
}

func NewTCPMessage(msg string, conn net.Conn, replyChan chan any) *TCPMessage {
	return &TCPMessage{
		DefaultMsg: shared.DefaultMsg{
			Msg:       msg,
			Payload:   nil,
			Source:    "TCP_SERVER",
			ReplyChan: replyChan, // No reply channel for TCP messages, normally
		},
		conn: conn,
	}
}

func (msg *TCPMessage) GetConn() net.Conn {
	return msg.conn
}
