package event_bus

import "roboserver/shared/data_structures"

// If an event has 0 subscribers, it is removed from the EventBus.
// Publishing to an event with no subscribers is a no-op.
type EventBus_t struct {
	subscriptions *data_structures.SafeMap[string, *data_structures.SafeSet[Subscriber]]                    // event type -> subscribers
	handlers      *data_structures.SafeMap[Subscriber, *data_structures.SafeMap[string, SubscriberHandler]] // Subscriber -> event -> handler function
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
	GetDataPtr() *interface{} // Returns pointer to data
}

// DefaultEvent is a simple implementation of the Event interface
type DefaultPtrEvent struct { // For larger data, use pointers to avoid copying
	Type string
	Data *interface{}
}

type DefaultEvent struct { // For smaller data, use values to avoid pointer dereferencing
	Type string
	Data interface{}
}
