package tcp_server

import "roboserver/shared"

func (s *TCPServer_t) validateRobot(ip string) shared.RobotHandler {
	handler, err := s.rm.GetHandler("", ip)
	if err != nil {
		shared.DebugPrint("tcp_server/utils.go", 10, "No robot handler found for IP: %s", ip)
		return nil
	}
	return handler
}
