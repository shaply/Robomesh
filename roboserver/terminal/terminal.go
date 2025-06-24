package terminal

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"roboserver/shared/robot_manager"
	"strings"
)

/* For debugging and testing purposes, this terminal server allows direct interaction with robots via TCP connections. */
func Start(ctx context.Context, robotHandler *robot_manager.RobotHandler, cancel context.CancelFunc) error {
	port := os.Getenv("TERMINAL_PORT")
	if port == "" {
		log.Fatal("TERMINAL_PORT environment variable is not set")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal("Error starting terminal server:", err)
	}
	defer listener.Close()

	log.Printf("Terminal server listening on port %s", port)

	go func() {
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
			go handleConnection(ctx, conn, robotHandler, cancel) // Handle each connection in a separate goroutine
		}
	}()

	<-ctx.Done() // wait for cancellation
	log.Println("Shutting down terminal server...")
	if err := listener.Close(); err != nil {
		return fmt.Errorf("error shutting down terminal server: %w", err)
	}
	log.Println("Terminal server has shut down gracefully.")
	return nil
}

// handleConnection handles an individual TCP connection for the terminal server.
func handleConnection(ctx context.Context, conn net.Conn, robotHandler *robot_manager.RobotHandler, cancel context.CancelFunc) {
	defer conn.Close()
	log.Printf("Handling connection from %s", conn.RemoteAddr())
	var args []string
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, closing connection")
			return
		default:
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				log.Println("Error reading from connection:", err)
				return
			}
			if n == 0 {
				log.Println("Connection closed by client")
				return
			}

			args = strings.Fields(string(buffer[:n]))
		}

		if len(args) == 0 {
			continue
		} else if args[0] == "exit" || args[0] == "quit" {
			log.Println("Exiting terminal session")
			return
		} else if args[0] == "stop" {
			if len(args) == 2 {
				if args[1] == "program" {
					cancel()
				}
			}
		} else if args[0] == "list" {
			if (len(args) == 2 && args[1] == "robots") || len(args) == 1 {
				robots := robotHandler.GetRobots()
				if len(robots) == 0 {
					conn.Write([]byte("No robots registered.\n"))
				} else {
					for _, robot := range robots {
						conn.Write([]byte(robot.String()))
						conn.Write([]byte("\n"))
					}
				}
			} else {
				conn.Write([]byte("Usage: list robots\n"))
			}
		}
	}
}
