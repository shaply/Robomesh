package tcp_server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"roboserver/shared/robot_manager"
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

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		message := strings.TrimSpace(scanner.Text())
		log.Printf("Received message: %s from ip %s", message, conn.RemoteAddr().String())

		s.processMessage(conn, message)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
	}
}

func (s *TCPServer) processMessage(conn net.Conn, message string) {
	args := strings.Fields(message)
	if len(args) == 0 {
		log.Println("Received empty message, ignoring.")
		return
	}

	switch args[0] {
	case "REGISTER":
		handleRegister(s, conn, args)
	case "UNREGISTER":
		handleUnregister(s, conn, args)
	default:
		log.Printf("Unknown command: %s", args[0])
		conn.Write([]byte("ERROR UNKNOWN_COMMAND\n"))
	}
}
