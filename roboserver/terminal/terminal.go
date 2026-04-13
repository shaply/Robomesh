package terminal

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/shared"
	"strings"
)

/* For debugging and testing purposes, this terminal server allows direct interaction via TCP connections. */
func Start(ctx context.Context, bus comms.Bus, db database.DBManager, cancel context.CancelFunc) error {
	port := shared.AppConfig.Server.TerminalPort

	// Bind to localhost only — the terminal has no authentication and provides
	// full admin access (shutdown, accept/reject registrations, list robots).
	// Exposing it on all interfaces would be a critical security issue.
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("error starting terminal server: %w", err)
	}
	defer listener.Close()

	shared.DebugPrint("Terminal server listening on port %d", port)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					shared.DebugPrint("Error accepting connection: %v", err)
					continue
				}
			}
			shared.DebugPrint("Accepted terminal connection from %s", conn.RemoteAddr())
			go handleConnection(ctx, conn, bus, db, cancel)
		}
	}()

	<-ctx.Done()
	shared.DebugPrint("Shutting down terminal server...")
	if err := listener.Close(); err != nil {
		return fmt.Errorf("error shutting down terminal server: %w", err)
	}
	shared.DebugPrint("Terminal server has shut down gracefully.")
	return nil
}

// handleConnection handles an individual TCP connection for the terminal server using the command registry.
func handleConnection(ctx context.Context, conn net.Conn, bus comms.Bus, db database.DBManager, cancel context.CancelFunc) {
	defer conn.Close()
	shared.DebugPrint("Handling terminal connection from %s", conn.RemoteAddr())

	cmdCtx := &CommandContext{
		Conn:          conn,
		DB:            db,
		Bus:           bus,
		Cancel:        cancel,
		Subscriptions: make(map[string]func()),
	}

	// Ensure all event subscriptions are cancelled when the connection closes
	defer func() {
		for topic, cancelFn := range cmdCtx.Subscriptions {
			cancelFn()
			delete(cmdCtx.Subscriptions, topic)
		}
	}()

	conn.Write([]byte("=== Robomesh Terminal ===\n"))
	conn.Write([]byte("Type 'help' for available commands.\n"))
	conn.Write([]byte("> "))

	scanner := bufio.NewScanner(conn)

	for {
		select {
		case <-ctx.Done():
			shared.DebugPrint("Context cancelled, closing terminal connection")
			conn.Write([]byte("\nTerminal session ended.\n"))
			return
		default:
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

			err := DefaultRegistry.ExecuteCommand(cmdCtx, command, commandArgs)
			if err != nil {
				if err.Error() == "exit" {
					return
				}
				conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			}

			conn.Write([]byte("> "))
		}
	}
}
