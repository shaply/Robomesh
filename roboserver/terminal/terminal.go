package terminal

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

/* For debugging and testing purposes, this terminal server allows direct interaction with robots via TCP connections. */
func Start(ctx context.Context, robotHandler robot_manager.RobotManager, cancel context.CancelFunc, eventBus event_bus.EventBus) error {
	port := os.Getenv("TERMINAL_PORT")
	if port == "" {
		shared.DebugPrint("TERMINAL_PORT environment variable is not set, using default port 9001")
		port = "9001"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("error starting terminal server: %w", err)
	}
	defer listener.Close()

	shared.DebugPrint("Terminal server listening on port %s", port)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return // Context cancelled, exit gracefully
				default:
					shared.DebugPrint("Error accepting connection: %v", err)
					continue
				}
			}
			shared.DebugPrint("Accepted terminal connection from %s", conn.RemoteAddr())
			go handleConnection(ctx, conn, robotHandler, cancel, eventBus) // Handle each connection in a separate goroutine
		}
	}()

	<-ctx.Done() // wait for cancellation
	shared.DebugPrint("Shutting down terminal server...")
	if err := listener.Close(); err != nil {
		return fmt.Errorf("error shutting down terminal server: %w", err)
	}
	shared.DebugPrint("Terminal server has shut down gracefully.")
	return nil
}

// handleConnection handles an individual TCP connection for the terminal server using the command registry.
func handleConnection(ctx context.Context, conn net.Conn, robotHandler robot_manager.RobotManager, cancel context.CancelFunc, eventBus event_bus.EventBus) {
	defer conn.Close()
	shared.DebugPrint("Handling terminal connection from %s", conn.RemoteAddr())

	// Create command context
	cmdCtx := &CommandContext{
		Conn:         conn,
		RobotManager: robotHandler,
		EventBus:     eventBus,
		Cancel:       cancel,
		Subscriber:   event_bus.NewSubscriber(),
	}

	// Send welcome message
	conn.Write([]byte("=== Robot Terminal ===\n"))
	conn.Write([]byte("Type 'help' for available commands.\n"))
	conn.Write([]byte("> "))

	// Use buffered scanner for better line handling
	scanner := bufio.NewScanner(conn)

	for {
		select {
		case <-ctx.Done():
			shared.DebugPrint("Context cancelled, closing terminal connection")
			conn.Write([]byte("\nTerminal session ended.\n"))
			return
		default:
			// Set read timeout to avoid blocking forever
			// conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					shared.DebugPrint("Error reading from terminal connection: %v", err)
				} else {
					shared.DebugPrint("Terminal connection closed by client")
				}
				return
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				conn.Write([]byte("> "))
				continue
			}

			args := strings.Fields(line)
			if len(args) == 0 {
				conn.Write([]byte("> "))
				continue
			}

			command := args[0]
			commandArgs := args[1:]

			// Execute command using registry
			err := DefaultRegistry.ExecuteCommand(cmdCtx, command, commandArgs)
			if err != nil {
				if err.Error() == "exit" {
					// Clean exit requested
					return
				}
				// Show error to user
				conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			}

			conn.Write([]byte("> "))
		}
	}
}
