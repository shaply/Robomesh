package terminal

import (
	"fmt"
	"roboserver/shared/event_bus"
)

func subscribeCommand(ctx *CommandContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: subscribe <event_type>")
	}

	eventType := args[0]
	ctx.EventBus.Subscribe(eventType, ctx.Subscriber, func(event event_bus.Event) {
		ctx.Conn.Write([]byte(fmt.Sprintf("\nEvent received: %s\n", event.GetType())))
		ctx.Conn.Write([]byte(fmt.Sprintf("Data: %v\n", event.GetData())))
	})
	ctx.Conn.Write([]byte(fmt.Sprintf("Subscribed to event type: %s\n", eventType)))
	return nil
}

func unsubscribeCommand(ctx *CommandContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: unsubscribe <event_type>")
	}

	eventType := args[0]
	ctx.EventBus.Unsubscribe(eventType, ctx.Subscriber)
	ctx.Conn.Write([]byte(fmt.Sprintf("Unsubscribed from event type: %s\n", eventType)))
	return nil
}

func publishCommand(ctx *CommandContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: publish <event_type> <data>")
	}

	eventType := args[0]
	data := args[1]

	event := event_bus.NewDefaultEvent(eventType, data)
	ctx.EventBus.Publish(event)
	ctx.Conn.Write([]byte("Published event\n"))
	return nil
}
