package shared

type Robot interface {
	ToJSON() string
	FromJSON(data string) error
	ToBaseRobot() BaseRobot
	FromBaseRobot(base BaseRobot) error
	IsOnline() bool
	String() string

	HandleMessage(message []byte) error
	HandleCommand(command string, args ...string) error
}
type BaseRobot struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	RobotType string `json:"type"`
	Status    string `json:"status"`
	DeviceID  string `json:"device_id"`
}
