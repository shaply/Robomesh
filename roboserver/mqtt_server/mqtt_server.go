package mqtt_server

import (
	"context"
	"roboserver/shared"
	"time"
)

type MQTTServer_t struct{}

func Start(ctx context.Context) error {
	shared.DebugPrint("MQTT server started (stub)")
	for {
		select {
		case <-ctx.Done():
			shared.DebugPrint("MQTT server shutting down...")
			return nil
		default:
			time.Sleep(1 * time.Second)
		}
	}
}
