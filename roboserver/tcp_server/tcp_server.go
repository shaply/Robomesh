package tcp_server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"roboserver/shared/robot_manager"
	"strconv"
	"strings"
)

type TCPServer struct {
	robotHandler *robot_manager.RobotHandler
	listener     net.Listener
}

func Start(ctx context.Context, robotHandler *robot_manager.RobotHandler) error {
	port := os.Getenv("TCP_PORT")
	if port == "" {
		log.Fatal("TCP_PORT environment variable is not set")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal("Error starting TCP server:", err)
	}
	defer listener.Close()

	s := &TCPServer{
		robotHandler: robotHandler,
		listener:     listener,
	}

	go func() {
		log.Printf("TCP server listening on port %s", port)
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return // Context cancelled, exit gracefully
				default:
					continue
				}
			}
			log.Printf("Accepted connection from %s", conn.RemoteAddr())
			go s.handleConnection(conn) // Handle each connection in a separate goroutine
		}
	}()
	<-ctx.Done() // wait for cancellation
	log.Println("Shutting down TCP server...")
	if err := listener.Close(); err != nil {
		log.Println("Error shutting down TCP server:", err)
		return fmt.Errorf("error shutting down TCP server: %w", err)
	}
	log.Println("TCP server has shut down gracefully.")
	return nil
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	data, err := io.ReadAll(conn)
	if err != nil {
		log.Printf("Error reading from connection: %v", err)
		return
	}

	message := string(data)
	log.Printf("Received message: %s from ip %s", message, conn.RemoteAddr().String())

	// Here you can add logic to handle the message, e.g., parse it, store it, etc.
	args := strings.Fields(message)
	switch args[0] {
	case "REGISTER":
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
		return
	case "UNREGISTER":
		if len(args) < 2 {
			ip := conn.RemoteAddr().String()
			log.Printf("Unregistering robot with IP: %s", ip)
			if err := s.robotHandler.UnregisterRobotByIP(ip); err != nil {
				log.Printf("Error unregistering robot: %v", err)
				return
			}
			log.Printf("Robot with IP %s unregistered successfully", ip)
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
		} else {
			log.Println("Invalid UNREGISTER command format. Expected: UNREGISTER [<id>]")
			return
		}
	default:
		log.Printf("Unknown command: %s", args[0])
	}
}
