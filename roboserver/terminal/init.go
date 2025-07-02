package terminal

// Auto-register commands using init()
func init() {
	RegisterCommand("list", "List registered robots", "list [robots]", listRobotsCommand)
	RegisterCommand("stop", "Stop the program or robot", "stop program|<robot_id>", stopCommand)
	RegisterCommand("help", "Show available commands", "help [command]", helpCommand)
	RegisterCommand("status", "Get robot status", "status <robot_id>", statusCommand)
	RegisterCommand("exit", "Exit terminal session", "exit", exitCommand)
	RegisterCommand("quit", "Exit terminal session", "quit", quitCommand)
}
