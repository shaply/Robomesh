package terminal

import (
	"context"
	"fmt"
)

// listActiveCommand lists all currently active robots from Redis.
func listActiveCommand(ctx *CommandContext, args []string) error {
	rds := ctx.DB.Redis()
	if rds == nil {
		ctx.Conn.Write([]byte("Redis not available.\n"))
		return nil
	}

	robots, err := rds.GetAllActiveRobots(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get active robots: %w", err)
	}

	if len(robots) == 0 {
		ctx.Conn.Write([]byte("No active robots.\n"))
		return nil
	}

	ctx.Conn.Write([]byte("Active robots:\n"))
	for _, r := range robots {
		ctx.Conn.Write([]byte(fmt.Sprintf("  %s  type=%s  ip=%s  pid=%d\n", r.UUID, r.DeviceType, r.IP, r.PID)))
	}
	return nil
}

// listRegisteredCommand lists all robots from the PostgreSQL registry.
func listRegisteredCommand(ctx *CommandContext, args []string) error {
	pg := ctx.DB.Postgres()
	if pg == nil {
		ctx.Conn.Write([]byte("PostgreSQL not available.\n"))
		return nil
	}

	robots, err := pg.GetAllRobots(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get registered robots: %w", err)
	}

	if len(robots) == 0 {
		ctx.Conn.Write([]byte("No registered robots.\n"))
		return nil
	}

	ctx.Conn.Write([]byte("Registered robots:\n"))
	for _, r := range robots {
		bl := ""
		if r.IsBlacklisted {
			bl = " [BLACKLISTED]"
		}
		ctx.Conn.Write([]byte(fmt.Sprintf("  %s  type=%s%s\n", r.UUID, r.DeviceType, bl)))
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

	ctx.Conn.Write([]byte(fmt.Sprintf("Stopping robot %s...\n", args[0])))
	// TODO: could remove from Redis to trigger cleanup
	return nil
}

func helpCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 {
		ctx.Conn.Write([]byte("Available commands:\n"))
		for _, cmd := range DefaultRegistry.ListCommands() {
			ctx.Conn.Write([]byte(fmt.Sprintf("  %-12s - %s\n", cmd.Name, cmd.Description)))
		}
		ctx.Conn.Write([]byte("\nUse 'help <command>' for detailed usage.\n"))
		return nil
	}

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
		return fmt.Errorf("usage: status <uuid>")
	}

	uuid := args[0]
	rds := ctx.DB.Redis()
	if rds == nil {
		return fmt.Errorf("redis not available")
	}

	active, err := rds.GetActiveRobot(context.Background(), uuid)
	if err != nil {
		ctx.Conn.Write([]byte(fmt.Sprintf("Robot %s: offline\n", uuid)))
		return nil
	}

	ctx.Conn.Write([]byte(fmt.Sprintf("Robot %s: online  ip=%s  type=%s  pid=%d\n",
		uuid, active.IP, active.DeviceType, active.PID)))
	return nil
}

func exitCommand(ctx *CommandContext, args []string) error {
	ctx.Conn.Write([]byte("Goodbye!\n"))
	return fmt.Errorf("exit")
}

func quitCommand(ctx *CommandContext, args []string) error {
	return exitCommand(ctx, args)
}

// pendingCommand lists all robots awaiting registration approval.
func pendingCommand(ctx *CommandContext, args []string) error {
	rds := ctx.DB.Redis()
	if rds == nil {
		ctx.Conn.Write([]byte("Redis not available.\n"))
		return nil
	}

	pending, err := rds.GetAllPendingRobots(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get pending robots: %w", err)
	}

	if len(pending) == 0 {
		ctx.Conn.Write([]byte("No pending registrations.\n"))
		return nil
	}

	ctx.Conn.Write([]byte("Pending registrations:\n"))
	for _, r := range pending {
		ctx.Conn.Write([]byte(fmt.Sprintf("  %s  type=%s  ip=%s  key=%s...\n",
			r.UUID, r.DeviceType, r.IP, truncate(r.PublicKey, 16))))
	}
	return nil
}

// acceptCommand accepts a pending robot registration.
func acceptCommand(ctx *CommandContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: accept <uuid>")
	}
	uuid := args[0]
	rds := ctx.DB.Redis()
	if rds == nil {
		return fmt.Errorf("redis not available")
	}

	_, err := rds.GetPendingRobot(context.Background(), uuid)
	if err != nil {
		return fmt.Errorf("no pending registration found for %s", uuid)
	}

	if err := ctx.Bus.PublishRegistrationResponse(context.Background(), uuid, true); err != nil {
		return fmt.Errorf("failed to accept: %w", err)
	}

	ctx.Conn.Write([]byte(fmt.Sprintf("Accepted robot %s\n", uuid)))
	return nil
}

// rejectCommand rejects a pending robot registration.
func rejectCommand(ctx *CommandContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: reject <uuid>")
	}
	uuid := args[0]
	rds := ctx.DB.Redis()
	if rds == nil {
		return fmt.Errorf("redis not available")
	}

	_, err := rds.GetPendingRobot(context.Background(), uuid)
	if err != nil {
		return fmt.Errorf("no pending registration found for %s", uuid)
	}

	if err := ctx.Bus.PublishRegistrationResponse(context.Background(), uuid, false); err != nil {
		return fmt.Errorf("failed to reject: %w", err)
	}

	ctx.Conn.Write([]byte(fmt.Sprintf("Rejected robot %s\n", uuid)))
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
