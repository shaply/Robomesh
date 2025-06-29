package main

import (
	"context"
	"log"
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

	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get environment variables
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env files: %v", err)
		return
	}

	var wg sync.WaitGroup

	log.Println("Server is running on the following IPs:")
	// Print local IPs
	localIPs := shared.GetLocalIPs()
	for _, ip := range localIPs {
		log.Printf("%s\n", ip)
	}

	// Initialize robot manager
	robotManager := robot_manager.NewRobotHandler()
	if robotManager == nil {
		log.Fatal("Failed to initialize robot manager")
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
		log.Println("Context cancelled, shutting down servers...")
	case <-sigs:
		log.Println("Received termination signal, shutting down...")
		cancel() // cancel the context to stop the server gracefully
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All servers have shut down gracefully.")
	case <-time.After(20 * time.Second):
		log.Println("Timeout waiting for servers to shut down, forcing exit.")
	}
}
