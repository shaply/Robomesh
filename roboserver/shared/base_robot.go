package shared

import (
	"encoding/json"
	"fmt"
)

// Constructor functions
func NewBaseRobot(deviceID string, ip string, robotType RobotType, status string, battery byte, lastSeen int64, authToken string) *BaseRobot {
	return &BaseRobot{
		DeviceID:  deviceID,
		IP:        ip,
		RobotType: robotType,
		Status:    status,
		Battery:   battery,
		LastSeen:  lastSeen,
		AuthToken: authToken,
	}
}

func NewBaseRobotHandler(robot Robot, msg_chan chan Msg) *BaseRobotHandler {
	return &BaseRobotHandler{
		Robot:   robot,
		MsgChan: msg_chan, // Example buffer size, adjust as needed
	}
}

func NewBaseRobotConnHandler(deviceId string, ip string, handler RobotHandler) *BaseRobotConnHandler {
	return &BaseRobotConnHandler{
		DeviceID:       deviceId,
		IP:             ip,
		Handler:        handler,
		DisconnectChan: make(chan bool),
	}
}

// JSON serialization methods
func (br *BaseRobot) ToJSON() string {
	data, err := json.Marshal(br)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// Conversion methods
func (br *BaseRobot) GetBaseRobot() BaseRobot {
	return *br
}

func (br *BaseRobot) GetDeviceID() string {
	return br.DeviceID
}

func (br *BaseRobot) GetIP() string {
	return br.IP
}

// Status checking method
func (br *BaseRobot) IsOnline() bool {
	return br.Status == "online" || br.Status == "connected" || br.Status == "active"
}

func (br *BaseRobot) String() string {
	return fmt.Sprintf("Robot(DeviceID: %s, RobotType: %s, IP: %s, Status: %s, Battery: %d%%, LastSeen: %d)",
		br.DeviceID, br.RobotType, br.IP, br.Status, br.Battery, br.LastSeen)
}

func (br *BaseRobotHandler) GetRobot() Robot {
	return br.Robot
}
func (br *BaseRobotHandler) SendMsg(msg Msg) error {
	if br.MsgChan == nil {
		return ErrMsgChannelUninitialized
	}
	<-br.MsgChan
	return ErrMsgUnknownType
}
func (br *BaseRobotHandler) GetDeviceID() string {
	return br.Robot.GetDeviceID()
}
func (br *BaseRobotHandler) GetIP() string {
	return br.Robot.GetIP()
}

func (brc *BaseRobotConnHandler) Start() error {
	// Implement the logic to start the connection handling routine.
	// This should be an indefinite loop that processes messages from the MsgChan
	// and communicates with the robot.
	return nil
}

func (brc *BaseRobotConnHandler) Stop() error {
	// Implement the logic to stop the connection and clean up resources.
	// This should close the DisconnectChan and any other resources used.
	SafeClose(brc.DisconnectChan)
	return nil
}

func (brc *BaseRobotConnHandler) GetHandler() RobotHandler {
	return brc.Handler
}

func (brc *BaseRobotConnHandler) GetDisconnectChannel() chan bool {
	return brc.DisconnectChan
}
