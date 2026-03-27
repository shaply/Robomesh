package terminal

func init() {
	RegisterCommand("list", "List active robots (from Redis)", "list", listActiveCommand)
	RegisterCommand("robots", "List registered robots (from PostgreSQL)", "robots", listRegisteredCommand)
	RegisterCommand("pending", "List pending robot registrations", "pending", pendingCommand)
	RegisterCommand("accept", "Accept a pending robot registration", "accept <uuid>", acceptCommand)
	RegisterCommand("reject", "Reject a pending robot registration", "reject <uuid>", rejectCommand)
	RegisterCommand("stop", "Stop the program or robot", "stop program|<robot_id>", stopCommand)
	RegisterCommand("help", "Show available commands", "help [command]", helpCommand)
	RegisterCommand("status", "Get robot status", "status <uuid>", statusCommand)
	RegisterCommand("exit", "Exit terminal session", "exit", exitCommand)
	RegisterCommand("quit", "Exit terminal session", "quit", quitCommand)
	RegisterCommand("subscribe", "Subscribe to robot events", "subscribe <event_type>", subscribeCommand)
	RegisterCommand("unsubscribe", "Unsubscribe from robot events", "unsubscribe <event_type>", unsubscribeCommand)
	RegisterCommand("publish", "Publish an event to robots", "publish <event_type> <data>", publishCommand)
}
