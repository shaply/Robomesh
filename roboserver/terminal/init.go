package terminal

// Auto-register commands using init()
func init() {
	RegisterCommand("list", "List registered robots", "list [robots]", listRobotsCommand)
	RegisterCommand("list_registering", "List registering robots", "list_registering", listRegisteringCommand)
	RegisterCommand("stop", "Stop the program or robot", "stop program|<robot_id>", stopCommand)
	RegisterCommand("help", "Show available commands", "help [command]", helpCommand)
	RegisterCommand("status", "Get robot status", "status <robot_id>", statusCommand)
	RegisterCommand("exit", "Exit terminal session", "exit", exitCommand)
	RegisterCommand("quit", "Exit terminal session", "quit", quitCommand)
	RegisterCommand("subscribe", "Subscribe to robot events", "subscribe <event_type>", subscribeCommand)
	RegisterCommand("unsubscribe", "Unsubscribe from robot events", "unsubscribe <event_type>", unsubscribeCommand)
	RegisterCommand("publish", "Publish an event to robots", "publish <event_type> <data>", publishCommand)
}
