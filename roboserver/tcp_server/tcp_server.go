package tcp_server

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"roboserver/shared"
	"roboserver/shared/event_bus"
	"roboserver/shared/robot_manager"
	"strings"
)

type TCPServer_t struct {
	rm           robot_manager.RobotManager
	eb           event_bus.EventBus // Event bus for handling events
	listener     net.Listener
	main_context context.Context // The main context to listen for cancellation
}

func Start(ctx context.Context, robotHandler robot_manager.RobotManager, eventBus event_bus.EventBus) error {
	port := os.Getenv("TCP_PORT")
	if port == "" {
		shared.DebugPanic("TCP_PORT environment variable is not set")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		shared.DebugPanic("Error starting TCP server:", err)
	}
	defer listener.Close()

	s := &TCPServer_t{
		rm:           robotHandler,
		eb:           eventBus,
		listener:     listener,
		main_context: ctx,
	}

	go func() {
		shared.DebugPrint("TCP server listening on port %s", port)
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
			shared.DebugPrint("Accepted connection from %s", conn.RemoteAddr())
			go s.handleConnection(conn) // Handle each connection in a separate goroutine
		}
	}()
	<-ctx.Done() // wait for cancellation
	shared.DebugPrint("Shutting down TCP server...")
	if err := listener.Close(); err != nil {
		shared.DebugPrint("Error shutting down TCP server:", err)
		return fmt.Errorf("error shutting down TCP server: %w", err)
	}
	shared.DebugPrint("TCP server has shut down gracefully.")
	return nil
}

func (s *TCPServer_t) handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		message := strings.TrimSpace(scanner.Text())
		shared.DebugPrint("Received message: %s from ip %s", message, conn.RemoteAddr().(*net.TCPAddr).IP.String())

		s.processMessage(conn, message)
	}

	if err := scanner.Err(); err != nil {
		shared.DebugPrint("Error reading from connection: %v", err)
	}
}

func (s *TCPServer_t) processMessage(conn net.Conn, message string) {
	args := strings.Fields(message)
	if len(args) == 0 {
		shared.DebugPrint("Received empty message, ignoring.")
		return
	}

	switch args[0] {
	case "REGISTER":
		handleRegister(s, conn, args)
	case "TRANSFER":
		handleTransfer(s, conn, args[0])
	default:
		handleDefault(s, conn, message)
	}
}
