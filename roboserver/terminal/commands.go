// terminal/commands.go
package terminal

import (
	"context"
	"fmt"
	"net"
	"roboserver/shared/robot_manager"
)

// CommandFunc represents a terminal command function
type CommandFunc func(ctx *CommandContext, args []string) error

// CommandInfo holds metadata about a command
type CommandInfo struct {
	Name        string
	Description string
	Usage       string
	Handler     CommandFunc
}

// CommandContext provides context for command execution
type CommandContext struct {
	Conn         net.Conn
	RobotManager *robot_manager.RobotManager
	Cancel       context.CancelFunc
}

// CommandRegistry holds all registered commands
type CommandRegistry struct {
	commands map[string]*CommandInfo
}

var DefaultRegistry = &CommandRegistry{
	commands: make(map[string]*CommandInfo),
}

// RegisterCommand registers a new command
func RegisterCommand(name, description, usage string, handler CommandFunc) {
	DefaultRegistry.commands[name] = &CommandInfo{
		Name:        name,
		Description: description,
		Usage:       usage,
		Handler:     handler,
	}
}

// GetCommand retrieves a command by name
func (r *CommandRegistry) GetCommand(name string) (*CommandInfo, bool) {
	cmd, exists := r.commands[name]
	return cmd, exists
}

// ListCommands returns all registered commands
func (r *CommandRegistry) ListCommands() []*CommandInfo {
	var commands []*CommandInfo
	for _, cmd := range r.commands {
		commands = append(commands, cmd)
	}
	return commands
}

// ExecuteCommand executes a command by name
func (r *CommandRegistry) ExecuteCommand(ctx *CommandContext, name string, args []string) error {
	cmd, exists := r.GetCommand(name)
	if !exists {
		return fmt.Errorf("unknown command: %s", name)
	}

	return cmd.Handler(ctx, args)
}
