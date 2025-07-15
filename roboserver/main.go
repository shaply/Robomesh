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
	"fmt"
	"os"
	"os/signal"
	"roboserver/database"
	"roboserver/http_server"
	"roboserver/mqtt_server"
	"roboserver/shared"
	"roboserver/shared/event_bus"
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
		panic(fmt.Sprintf("Error loading .env file: %v", err))
	}
	shared.InitConfig()

	var wg sync.WaitGroup

	shared.DebugPrint("Server is running on the following IPs:")
	// Print local IPs
	localIPs := shared.GetLocalIPs()
	for _, ip := range localIPs {
		shared.DebugPrint("%s", ip)
	}

	// Initialize event bus
	eventBus := event_bus.NewEventBus()
	if eventBus == nil {
		panic("Failed to initialize event bus")
	}

	// Initialize database manager
	dbManager, err := database.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize databases: %v", err))
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		dbManager.Stop() // Ensure databases are stopped on context cancellation
	}()

	// Initialize robot manager
	robotManager := robot_manager.NewRobotManager(ctx, dbManager, eventBus)
	if robotManager == nil {
		panic("Failed to initialize robot manager")
	}

	// TODO: Pass dbManager to components that need database access
	// For now, components can access the database manager when their signatures are updated
	// Example: http_server.Start(ctx, robotManager, dbManager)
	_ = dbManager // Prevent unused variable error until components are updated

	// Start terminal server (for debugging purposes)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := terminal.Start(ctx, robotManager, cancel, eventBus); err != nil {
			shared.DebugError(err)
			cancel()
		}
	}()

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http_server.Start(ctx, robotManager, eventBus); err != nil {
			shared.DebugError(err)
			cancel()
		}
	}()

	// Start MQTT server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := mqtt_server.Start(ctx, robotManager); err != nil {
			shared.DebugError(err)
			cancel()
		}
	}()

	// Start TCP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tcp_server.Start(ctx, robotManager, eventBus); err != nil {
			shared.DebugError(err)
			cancel()
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		shared.DebugPrint("Context cancelled, shutting down servers...")
	case <-sigs:
		shared.DebugPrint("Received termination signal, shutting down...")
	}

	cancel() // cancel the context to stop the server gracefully

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

// Example of how to integrate database operations with robot management:
//
// Robot Registration with Database Persistence:
//   func (rm *RobotManager) RegisterRobotWithDB(deviceID string, ip string, robotType shared.RobotType, dbManager *database.DatabaseManager) error {
//       // Register robot in memory
//       if err := rm.RegisterRobot(deviceID, ip, robotType); err != nil {
//           return err
//       }
//
//       // Persist to database
//       if dbManager != nil {
//           collection, err := dbManager.GetMongoDB().GetCollection("robots")
//           if err != nil {
//               return err
//           }
//
//           robotDoc := bson.M{
//               "device_id": deviceID,
//               "ip": ip,
//               "robot_type": robotType,
//               "status": "online",
//               "registered_at": time.Now(),
//           }
//
//           _, err = collection.InsertOne(context.Background(), robotDoc)
//           return err
//       }
//       return nil
//   }
//
// Robot Status Updates:
//   func updateRobotStatus(deviceID string, status string, dbManager *database.DatabaseManager) error {
//       collection, err := dbManager.GetMongoDB().GetCollection("robots")
//       if err != nil {
//           return err
//       }
//
//       filter := bson.M{"device_id": deviceID}
//       update := bson.M{
//           "$set": bson.M{
//               "status": status,
//               "last_updated": time.Now(),
//           },
//       }
//
//       _, err = collection.UpdateOne(context.Background(), filter, update)
//       return err
//   }
//
// Sensor Data Storage:
//   func storeSensorData(deviceID string, sensorData interface{}, dbManager *database.DatabaseManager) error {
//       collection, err := dbManager.GetMongoDB().GetCollection("sensor_data")
//       if err != nil {
//           return err
//       }
//
//       document := bson.M{
//           "device_id": deviceID,
//           "timestamp": time.Now(),
//           "data": sensorData,
//       }
//
//       _, err = collection.InsertOne(context.Background(), document)
//       return err
//   }
