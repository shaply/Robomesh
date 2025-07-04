// Package main is the entry point for the Robomesh server application.
//
// This server provides a comprehensive robot management platform with multiple
// communication protocols (HTTP, MQTT, TCP) and a command-line interface for
// debugging and administration. The server coordinates robot connections,
// manages robot state, and provides APIs for robot control and monitoring.
//
// Architecture Overview:
//
// The server consists of several key components:
// - RobotManager: Central coordinator for robot lifecycle and state
// - HTTP Server: REST API for web-based robot control and monitoring
// - MQTT Server: Message broker for IoT-style robot communication
// - TCP Server: Low-level socket communication for robots
// - Terminal: Interactive command-line interface for debugging
//
// Supported Robot Types:
// - Proximity Sensors: Distance measurement and obstacle detection
// - Door Openers: Smart door control and access management
// - Trash Cans: Waste management and capacity monitoring
// - (Extensible via robot type registration)
//
// Configuration:
// The server uses environment variables for configuration, loaded from a .env file:
// - DEBUG: Enable verbose logging and debug features
// - Additional configuration can be added in shared/config.go
//
// Graceful Shutdown:
// The server supports graceful shutdown via SIGINT/SIGTERM signals, ensuring
// all robot connections are properly closed and resources are cleaned up.
package main

import (
	"context"
	"os"
	"os/signal"
	"roboserver/http_server"
	"roboserver/mqtt_server"
	"roboserver/shared"
	"roboserver/shared/robot_manager"
	"roboserver/tcp_server"
	"roboserver/terminal"
	"sync"
	"syscall"
	"time"

	_ "roboserver/robots" // Import all robots to register them

	"github.com/joho/godotenv"
)

// main is the application entry point that initializes and starts all server components.
//
// The function performs these operations in sequence:
// 1. Sets up context for coordinated shutdown across all components
// 2. Loads environment configuration from .env file
// 3. Displays available network interfaces for robot connection
// 4. Initializes the robot manager for centralized robot coordination
// 5. Starts all server components (terminal, HTTP, MQTT, TCP) in separate goroutines
// 6. Waits for shutdown signals and coordinates graceful termination
//
// Server Components:
//
// Terminal Server:
// - Interactive command-line interface for debugging and robot control
// - Provides direct access to robot manager functions
// - Useful for development and troubleshooting
//
// HTTP Server:
// - REST API for web-based robot control and monitoring
// - JSON-based communication suitable for web applications
// - Provides robot discovery, status, and control endpoints
//
// MQTT Server:
// - Message broker for IoT-style robot communication
// - Supports publish/subscribe patterns for robot coordination
// - Suitable for event-driven robot interactions
//
// TCP Server:
// - Low-level socket communication for high-performance robot control
// - Binary protocol for minimal latency and overhead
// - Direct robot registration and message passing
//
// Error Handling:
// - Critical initialization failures cause immediate shutdown
// - Server startup errors are logged but don't prevent other servers from starting
// - Graceful shutdown ensures all resources are properly cleaned up
//
// Shutdown Behavior:
// - Responds to SIGINT (Ctrl+C) and SIGTERM signals
// - Cancels context to signal all components to shut down
// - Waits up to 60 seconds for graceful shutdown
// - Forces exit if graceful shutdown times out
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get environment variables
	err := godotenv.Load(".env")
	if err != nil {
		shared.DebugPrint("Error loading .env files: %v", err)
		return
	}
	shared.InitConfig()

	var wg sync.WaitGroup

	shared.DebugPrint("Server is running on the following IPs:")
	// Print local IPs
	localIPs := shared.GetLocalIPs()
	for _, ip := range localIPs {
		shared.DebugPrint("%s", ip)
	}

	// Initialize robot manager
	robotManager := robot_manager.NewRobotManager(ctx)
	if robotManager == nil {
		shared.DebugPanic("Failed to initialize robot manager")
	}

	// Start terminal server (for debugging purposes)
	wg.Add(1)
	go func() {
		defer wg.Done()
		terminal.Start(ctx, robotManager, cancel)
	}()

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		http_server.Start(ctx, robotManager)
	}()

	// Start MQTT server
	wg.Add(1)
	go func() {
		defer wg.Done()
		mqtt_server.Start(ctx, robotManager)
	}()

	// Start TCP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		tcp_server.Start(ctx, robotManager)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		shared.DebugPrint("Context cancelled, shutting down servers...")
	case <-sigs:
		shared.DebugPrint("Received termination signal, shutting down...")
		cancel() // cancel the context to stop the server gracefully
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		shared.DebugPrint("All servers have shut down gracefully.")
	case <-time.After(60 * time.Second):
		shared.DebugPrint("Timeout waiting for servers to shut down, forcing exit.")
	}
}
