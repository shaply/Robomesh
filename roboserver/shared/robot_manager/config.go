package robot_manager

var (
	// For the event bus registering robots
	REGISTERING_ROBOT_EVENT            = "robot_manager.registering_robot"
	HANDLE_REGISTERING_ROBOT_EVENT_FMT = "register.%s:%s:%s" // deviceID:ip:robotType
	ACCEPT_REGISTERING_ROBOT_RESPONSE  = true
)
