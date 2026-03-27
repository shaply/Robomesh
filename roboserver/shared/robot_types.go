package shared

// RobotType represents the category of robot and determines its handler script.
type RobotType string

// BaseRobot provides the fundamental state and metadata for all robot types.
type BaseRobot struct {
	DeviceID  string    `json:"device_id"`
	IP        string    `json:"ip,omitempty"`
	RobotType RobotType `json:"robot_type"`
	Status    string    `json:"status"`
	Battery   byte      `json:"battery,omitempty"`
	LastSeen  int64     `json:"last_seen,omitempty"`
}
