package tcp_server

import (
	"log"
	"net"
	"strconv"
)

func handleRegister(s *TCPServer, conn net.Conn, args []string) {
	if len(args) < 3 {
		conn.Write([]byte("ERROR REGISTER\n"))
		log.Println("Invalid REGISTER command format. Expected: REGISTER <robot_type> <device_id>")
		return
	}
	robotType := args[1]
	deviceID := args[2]
	log.Printf("Registering robot: %s with device ID: %s", robotType, deviceID)
	if err := s.robotHandler.RegisterRobot(conn.RemoteAddr().String(), robotType, deviceID); err != nil {
		conn.Write([]byte("ERROR REGISTER\n"))
		log.Printf("Error registering robot: %v", err)
		return
	}
	log.Printf("Robot registered successfully: %s (%s)", robotType, deviceID)
	conn.Write([]byte("OK REGISTER\n"))
}

func handleUnregister(s *TCPServer, conn net.Conn, args []string) {
	if len(args) < 2 {
		ip := conn.RemoteAddr().String()
		log.Printf("Unregistering robot with IP: %s", ip)
		if err := s.robotHandler.UnregisterRobotByIP(ip); err != nil {
			log.Printf("Error unregistering robot: %v", err)
			return
		}
		log.Printf("Robot with IP %s unregistered successfully", ip)
		return
	}

	id := args[1]
	log.Printf("Unregistering robot with ID: %s", id)
	intID, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("Invalid robot ID: %s. Error: %v", id, err)
		return
	}
	if err := s.robotHandler.UnregisterRobot(intID); err != nil {
		log.Printf("Error unregistering robot: %v", err)
		return
	}
	log.Printf("Robot with ID %s unregistered successfully", id)
}
