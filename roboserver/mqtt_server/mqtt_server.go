package mqtt_server

import (
	"context"
	"log"
	"time"
)

func Start(ctx context.Context) {
	log.Println("MQTT client started")
	for {
		select {
		case <-ctx.Done():
			log.Println("MQTT client shutting down...")
			return
		default:
			// Simulate polling or handling messages
			time.Sleep(2 * time.Second)
		}
	}
}
