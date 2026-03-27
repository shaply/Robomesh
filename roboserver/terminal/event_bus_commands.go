package terminal

import (
	"fmt"
)

func subscribeCommand(ctx *CommandContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: subscribe <event_type>")
	}

	eventType := args[0]
	cancel, err := ctx.Bus.SubscribeEvent(eventType, func(et string, data any) {
		ctx.Conn.Write([]byte(fmt.Sprintf("\nEvent received: %s\n", et)))
		ctx.Conn.Write([]byte(fmt.Sprintf("Data: %v\n", data)))
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Cancel any existing subscription for this event type
	if existing, ok := ctx.Subscriptions[eventType]; ok {
		existing()
	}
	ctx.Subscriptions[eventType] = cancel

	ctx.Conn.Write([]byte(fmt.Sprintf("Subscribed to event type: %s\n", eventType)))
	return nil
}

func unsubscribeCommand(ctx *CommandContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: unsubscribe <event_type>")
	}

	eventType := args[0]
	if cancel, ok := ctx.Subscriptions[eventType]; ok {
		cancel()
		delete(ctx.Subscriptions, eventType)
		ctx.Conn.Write([]byte(fmt.Sprintf("Unsubscribed from event type: %s\n", eventType)))
	} else {
		ctx.Conn.Write([]byte(fmt.Sprintf("Not subscribed to event type: %s\n", eventType)))
	}
	return nil
}

func publishCommand(ctx *CommandContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: publish <event_type> <data>")
	}

	eventType := args[0]
	data := args[1]

	if err := ctx.Bus.PublishEvent(eventType, data); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}
	ctx.Conn.Write([]byte("Published event\n"))
	return nil
}
