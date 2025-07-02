package proximity_sensor

import "roboserver/shared"

func NewRobotConnHandlerFunc(deviceId string, ip string) (shared.RobotConnHandler, error) {
	handler := &RobotConnHandler{
		*shared.NewBaseRobotConnHandler(deviceId, ip, NewRobotHandler(NewRobotInit(deviceId, ip))),
	}

	return handler, nil
}

type RobotConnHandler struct {
	shared.BaseRobotConnHandler
}

func (rc *RobotConnHandler) Start() error {
	<-rc.DisconnectChan
	shared.DebugPrint("Proximity sensor connection handler for device %s disconnected", rc.DeviceID)
	return nil
}
