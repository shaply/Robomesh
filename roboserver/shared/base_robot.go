package shared

import (
	"encoding/json"
	"fmt"
)

func NewBaseRobot(deviceID string, ip string, robotType RobotType, status string, battery byte, lastSeen int64) *BaseRobot {
	return &BaseRobot{
		DeviceID:  deviceID,
		IP:        ip,
		RobotType: robotType,
		Status:    status,
		Battery:   battery,
		LastSeen:  lastSeen,
	}
}

func (br *BaseRobot) ToJSON() string {
	data, err := json.Marshal(br)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (br *BaseRobot) String() string {
	return fmt.Sprintf("Robot(DeviceID: %s, RobotType: %s, IP: %s, Status: %s, Battery: %d%%, LastSeen: %d)",
		br.DeviceID, br.RobotType, br.IP, br.Status, br.Battery, br.LastSeen)
}
