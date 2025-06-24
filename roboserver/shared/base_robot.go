package shared

import (
	"encoding/json"
	"fmt"
)

// JSON serialization methods
func (br *BaseRobot) ToJSON() string {
	data, err := json.Marshal(br)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (br *BaseRobot) FromJSON(data string) error {
	return json.Unmarshal([]byte(data), br)
}

// Conversion methods
func (br *BaseRobot) ToBaseRobot() BaseRobot {
	return *br
}

func (br *BaseRobot) FromBaseRobot(base BaseRobot) error {
	br.ID = base.ID
	br.Name = base.Name
	br.IP = base.IP
	br.RobotType = base.RobotType
	br.Status = base.Status
	br.DeviceID = base.DeviceID
	return nil
}

// Status checking method
func (br *BaseRobot) IsOnline() bool {
	return br.Status == "online" || br.Status == "connected" || br.Status == "active"
}

func (br *BaseRobot) String() string {
	return fmt.Sprintf("Robot(ID: %d, Name: %s, Type: %s, IP: %s, Status: %s, DeviceID: %s)",
		br.ID, br.Name, br.RobotType, br.IP, br.Status, br.DeviceID)
}

// Message handling methods (basic implementations)
func (br *BaseRobot) HandleMessage(message []byte) error {
	// Basic implementation - you can override this in specific robot types
	fmt.Printf("Robot %d received message: %s\n", br.ID, string(message))
	return nil
}

func (br *BaseRobot) HandleCommand(command string, args ...string) error {
	// Basic implementation - you can override this in specific robot types
	switch command {
	case "ping":
		fmt.Printf("Robot %d: pong\n", br.ID)
		return nil
	case "status":
		fmt.Printf("Robot %d status: %s\n", br.ID, br.Status)
		return nil
	case "stop":
		br.Status = "stopped"
		fmt.Printf("Robot %d stopped\n", br.ID)
		return nil
	case "start":
		br.Status = "active"
		fmt.Printf("Robot %d started\n", br.ID)
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// Constructor function
func NewBaseRobot(name, robotType, ip, deviceID string) *BaseRobot {
	return &BaseRobot{
		ID:        0, // Will be set by RobotHandler
		Name:      name,
		IP:        ip,
		RobotType: robotType,
		Status:    "offline",
		DeviceID:  deviceID,
	}
}
