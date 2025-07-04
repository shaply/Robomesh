package shared

// Robot types
type RobotType string

const BASE_ROBOT_TYPE RobotType = "base_robot"

/*
To use this in personal robot, you embed BaseRobot in your robot struct
and implement the Robot interface. This allows you to add custom fields and methods while still using
then, when calling it, you can do type assertion to get the robot fields and methods.
For example:
```go

	type MyRobot struct {
		BaseRobot // Embed BaseRobot to inherit its fields and methods
		...Custom fields...
	}

func (r *MyRobot) ...methods... {}

robot := robot_manager.GetRobot("device_id", "")
myRobot, ok := robot.(*MyRobot)

	if !ok {
		// Handle error
	}

// Use myRobot which has fields of MyRobot and BaseRobot
```
*/
type Robot interface {
	ToJSON() string              // Convert robot to JSON string
	GetBaseRobot() BaseRobot     // Get the base robot structure
	GetDeviceID() string         // Get the unique device ID of the robot
	GetIP() string               // Get the IP address of the robot
	IsOnline() bool              // Check if the robot is online
	SetLastSeen(timestamp int64) // Set the last seen timestamp of the robot
	String() string              // Get a string representation of the robot
}

type BaseRobot struct {
	DeviceID  string    `json:"device_id"`           // Unique identifier for the robot's device, can be used for authentication and identification
	IP        string    `json:"ip,omitempty"`        // IP address of the robot, used for communication
	RobotType RobotType `json:"robot_type"`          // Type of the robot, e.g., "drone", "car", etc.
	Status    string    `json:"status"`              // Status of the robot, e.g., "online", "offline", "busy", etc.
	Battery   byte      `json:"battery,omitempty"`   // Optional field, can be omitted if not set
	LastSeen  int64     `json:"last_seen,omitempty"` // Optional field, can be omitted if not set
	AuthToken string    `json:"-"`                   // Authentication token, not serialized
}

type BaseRobotHandler struct {
	Robot      Robot    // The robot state, implements Robot interface
	MsgChan    chan Msg // Channel for receiving messages, implement message queue size yourself
	disconnect chan bool
}

type RobotHandler interface {
	GetRobot() Robot       // Get the robot state so services can use it
	SendMsg(msg Msg) error // Channel for receiving messages, implement message queue size yourself
	GetDeviceID() string
	GetIP() string
	GetDisconnectChannel() chan bool // Get the disconnect channel for the robot
	QuickAction()                    // Perform a quick action on the robot, e.g. status check, battery check, etc., for the http server, this method will most likely utilize send message to the robot
}

/*
Default type of message, but can be extended for specific robot types.
*/
type Msg interface {
	GetMsg() string         // Get the message content
	GetPayload() any        // Get the payload of the message
	GetSource() string      // Get the source of the message
	GetReplyChan() chan any // Get the reply channel for the message
}
type DefaultMsg struct {
	Msg       string   `json:"msg"`               // The message content
	Payload   any      `json:"payload,omitempty"` // Optional payload for the message
	Source    string   `json:"source,omitempty"`  // Optional source of the message
	ReplyChan chan any `json:"-"`                 // Channel for replies, not serialized
}

// NewRobotConnHandlerFunc is a function type that creates a new RobotConnHandler for the robot.
type NewRobotConnHandlerFunc func(deviceId string, ip string) (RobotConnHandler, error)

type BaseRobotConnHandler struct {
	DeviceID       string
	IP             string
	Handler        RobotHandler
	DisconnectChan chan bool // Channel that is closed when the connection is disconnected
}

/*
Manages the connections to robots.
*/
type RobotConnHandler interface {
	Start() error                    // Start the routine that handles messages in the message channel and communicates with the robot. Should be an indefinite loop till disconnection.
	Stop() error                     // Handles stopping the connection and cleaning up resources.
	GetHandler() RobotHandler        // Returns the RobotHandler that manages the output of robot's state and communication.
	GetDisconnectChannel() chan bool // Returns a channel that is closed when the connection is disconnected.
}
