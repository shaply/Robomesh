package mqtt_server

import (
	"context"
	"log"
	"roboserver/shared/robot_manager"
	"time"
)

type MQTTClient struct {
	robotHandler *robot_manager.RobotHandler
}

func Start(ctx context.Context, robotHandler *robot_manager.RobotHandler) {
	log.Println("MQTT client started")
	for {
		select {
		case <-ctx.Done():
			log.Println("MQTT client shutting down...")
			return
		default:
			// Simulate polling or handling messages
			time.Sleep(1 * time.Second)
		}
	}
}
