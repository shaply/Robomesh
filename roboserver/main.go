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
