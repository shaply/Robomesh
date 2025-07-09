package roboserver

import (
	"context"
	"roboserver/http_server"
	"roboserver/http_server/websocket"
	"roboserver/shared/robot_manager"
	"roboserver/tcp_server"
)

type RoboServer struct {
	// Core services
	RobotManager *robot_manager.RobotManager
	WSManager    *websocket.WSManager
	HTTPHandler  *http_server.HTTPServer
	TCPHandler   *tcp_server.TCPServer
	// mqttHandler  *mqtt_server.MQTTServer
	// DBManager *database.DBManager

	// Lifecycle
	Ctx    context.Context
	Cancel context.CancelFunc
}

func NewRoboServer() *RoboServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &RoboServer{
		Ctx:    ctx,
		Cancel: cancel,
	}
}
