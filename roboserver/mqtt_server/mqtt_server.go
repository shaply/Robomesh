package mqtt_server

import (
	"context"
	"roboserver/shared"
	"roboserver/shared/robot_manager"
	"time"
)

type MQTTServer_t struct {
	robotHandler robot_manager.RobotManager
}

func Start(ctx context.Context, robotHandler robot_manager.RobotManager) error {
	shared.DebugPrint("MQTT server started")
	for {
		select {
		case <-ctx.Done():
			shared.DebugPrint("MQTT server shutting down...")
			return nil
		default:
			// Simulate polling or handling messages
			time.Sleep(1 * time.Second)
		}
	}
}
