package mqtt_server

import (
	"context"
	"roboserver/shared"
	"roboserver/shared/robot_manager"
	"time"
)

type MQTTClient struct {
	robotHandler *robot_manager.RobotManager
}

func Start(ctx context.Context, robotHandler *robot_manager.RobotManager) {
	shared.DebugPrint("MQTT client started")
	for {
		select {
		case <-ctx.Done():
			shared.DebugPrint("MQTT client shutting down...")
			return
		default:
			// Simulate polling or handling messages
			time.Sleep(1 * time.Second)
		}
	}
}
