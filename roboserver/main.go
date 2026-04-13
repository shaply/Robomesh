package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/handler_engine"
	"roboserver/http_server"
	"roboserver/mqtt_server"
	"roboserver/shared"
	"roboserver/shared/event_bus"
	"roboserver/shared/utils"
	"roboserver/tcp_server"
	"roboserver/terminal"
	"roboserver/udp_server"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load .env for local development (non-fatal if missing)
	_ = godotenv.Load(".env")

	// Load structured config (config.yaml + env var overrides)
	if err := shared.LoadConfig("config.yaml"); err != nil {
		panic(fmt.Sprintf("Error loading configuration: %v", err))
	}

	var wg sync.WaitGroup

	shared.DebugPrint("Server is running on the following IPs:")
	localIPs := utils.GetLocalIPs()
	for _, ip := range localIPs {
		shared.DebugPrint("%s", ip)
	}

	// Initialize event bus
	eventBus := event_bus.NewEventBus()
	if eventBus == nil {
		panic("Failed to initialize event bus")
	}

	// Initialize database manager (PostgreSQL + Redis)
	dbManager, err := database.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize databases: %v", err))
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if dbManager != nil {
			dbManager.Stop()
		}
	}()

	// Initialize communication bus (wraps event bus + Redis pub/sub)
	var bus comms.Bus
	if dbManager != nil && dbManager.Redis() != nil {
		bus = comms.NewLocalBus(eventBus, dbManager.Redis())
	}

	// Start terminal server (for debugging)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := terminal.Start(ctx, bus, dbManager, cancel); err != nil {
			shared.DebugError(err)
			cancel()
		}
	}()

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http_server.Start(ctx, bus, dbManager); err != nil {
			shared.DebugError(err)
			cancel()
		}
	}()

	// Start MQTT server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := mqtt_server.Start(ctx, bus, dbManager); err != nil {
			shared.DebugError(err)
			cancel()
		}
	}()

	// Start TCP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tcp_server.Start(ctx, bus, dbManager); err != nil {
			shared.DebugError(err)
			cancel()
		}
	}()

	// Start UDP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := udp_server.Start(ctx, bus, dbManager); err != nil {
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

	cancel()

	// Stop all handler processes
	handler_engine.HandlerManager.StopAll("server_shutdown")

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
