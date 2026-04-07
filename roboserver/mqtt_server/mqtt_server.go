package mqtt_server

import (
	"context"
	"fmt"
	"roboserver/comms"
	"roboserver/database"
	"roboserver/shared"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

type MQTTServer_t struct {
	server *mqtt.Server
	bus    comms.Bus
	db     database.DBManager
}

func Start(ctx context.Context, bus comms.Bus, db database.DBManager) error {
	port := shared.AppConfig.Server.MQTTPort

	server := mqtt.New(&mqtt.Options{
		InlineClient: true,
	})

	// Allow all connections for now — robot auth is handled at the application layer
	// via topic-level ACLs and message verification, not MQTT CONNECT credentials.
	// For production, replace with a custom auth hook that verifies robot JWTs.
	if err := server.AddHook(new(auth.AllowHook), nil); err != nil {
		return fmt.Errorf("failed to add MQTT auth hook: %w", err)
	}

	// Add event bus bridge hook to forward MQTT publishes to the internal event bus
	if bus != nil {
		bridgeHook := &eventBusBridgeHook{bus: bus}
		if err := server.AddHook(bridgeHook, nil); err != nil {
			shared.DebugPrint("Failed to add MQTT event bus bridge: %v", err)
		}
	}

	// TCP listener
	tcp := listeners.NewTCP(listeners.Config{
		ID:      "mqtt-tcp",
		Address: fmt.Sprintf(":%d", port),
	})
	if err := server.AddListener(tcp); err != nil {
		return fmt.Errorf("failed to add MQTT TCP listener: %w", err)
	}

	// Start server
	go func() {
		shared.DebugPrint("Starting MQTT server on port %d", port)
		if err := server.Serve(); err != nil {
			shared.DebugPrint("MQTT server error: %v", err)
		}
	}()

	<-ctx.Done()
	shared.DebugPrint("Shutting down MQTT server...")
	if err := server.Close(); err != nil {
		shared.DebugPrint("Error shutting down MQTT server: %v", err)
		return fmt.Errorf("error shutting down MQTT server: %w", err)
	}
	shared.DebugPrint("MQTT server shut down gracefully")
	return nil
}
