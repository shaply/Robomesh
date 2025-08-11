// terminal/robot_commands.go
package terminal

import (
	"fmt"
)

// Command implementations
func listRobotsCommand(ctx *CommandContext, args []string) error {
	robots := ctx.RobotManager.GetRobots()
	if len(robots) == 0 {
		ctx.Conn.Write([]byte("No robots registered.\n"))
		return nil
	}

	ctx.Conn.Write([]byte("Registered robots:\n"))
	for _, robot := range robots {
		ctx.Conn.Write([]byte(fmt.Sprintf("  %s\n", robot.String())))
	}
	return nil
}

func listRegisteringCommand(ctx *CommandContext, args []string) error {
	registeringRobots := ctx.RobotManager.GetRegisteringRobots()
	if len(registeringRobots) == 0 {
		ctx.Conn.Write([]byte("No robots currently registering.\n"))
		return nil
	}

	ctx.Conn.Write([]byte("Registering robots:\n"))
	for _, robot := range registeringRobots {
		ctx.Conn.Write([]byte(fmt.Sprintf("  %v\n", robot)))
	}
	return nil
}

func stopCommand(ctx *CommandContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: stop program|<robot_id>")
	}

	if args[0] == "program" {
		ctx.Conn.Write([]byte("Stopping program...\n"))
		ctx.Cancel()
		return nil
	}

	// Stop specific robot logic here
	ctx.Conn.Write([]byte(fmt.Sprintf("Stopping robot %s...\n", args[0])))
	return nil
}

func helpCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 {
		// Show all commands
		ctx.Conn.Write([]byte("Available commands:\n"))
		for _, cmd := range DefaultRegistry.ListCommands() {
			ctx.Conn.Write([]byte(fmt.Sprintf("  %-10s - %s\n", cmd.Name, cmd.Description)))
		}
		ctx.Conn.Write([]byte("\nUse 'help <command>' for detailed usage.\n"))
		return nil
	}

	// Show specific command help
	cmd, exists := DefaultRegistry.GetCommand(args[0])
	if !exists {
		return fmt.Errorf("unknown command: %s", args[0])
	}

	ctx.Conn.Write([]byte(fmt.Sprintf("Command: %s\n", cmd.Name)))
	ctx.Conn.Write([]byte(fmt.Sprintf("Description: %s\n", cmd.Description)))
	ctx.Conn.Write([]byte(fmt.Sprintf("Usage: %s\n", cmd.Usage)))
	return nil
}

func statusCommand(ctx *CommandContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: status <robot_id>")
	}

	robotID := args[0]
	robot, err := ctx.RobotManager.GetRobot(robotID, "")
	if err != nil {
		return fmt.Errorf("robot not found: %s", robotID)
	}

	ctx.Conn.Write([]byte(fmt.Sprintf(robot.String())))
	return nil
}

func exitCommand(ctx *CommandContext, args []string) error {
	ctx.Conn.Write([]byte("Goodbye!\n"))
	return fmt.Errorf("exit") // Special error to signal exit
}

func quitCommand(ctx *CommandContext, args []string) error {
	return exitCommand(ctx, args)
}
