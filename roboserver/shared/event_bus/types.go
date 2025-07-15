package event_bus

import "roboserver/shared/event_bus/data_structures"

// If an event has 0 subscribers, it is removed from the EventBus.
// Publishing to an event with no subscribers is a no-op.
type EventBus_t struct {
	subscriptions *data_structures.SafeMap[string, *data_structures.Set[Subscriber]] // event type -> subscribers
	handlers      *data_structures.SafeMap[Subscriber, SubscriberHandler]            // Subscriber -> handler function
}

type Subscriber struct {
	ID string // This makes the struct comparable (functions are ignored for comparison)
	// Note: HandleEvent function is stored separately to avoid comparison issues
}

// SubscriberHandler maps subscriber IDs to their event handlers
type SubscriberHandler func(event Event)

type Event interface {
	GetType() string
	GetData() interface{}
}

// DefaultEvent is a simple implementation of the Event interface
type DefaultEvent struct {
	Type string
	Data interface{}
}
