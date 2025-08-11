package event_bus

// EventBus interface defines the contract for event-driven communication.
// Implementations provide thread-safe publish/subscribe operations for
// decoupled component communication with typed events and handlers.
type EventBus interface {
	// Subscribe registers a handler for events of a specific type.
	// Creates a new subscriber if nil is provided.
	// Returns the subscriber instance for later unsubscription.
	Subscribe(eventType string, subscriber *Subscriber, handler SubscriberHandler) *Subscriber

	// Unsubscribe removes a subscriber from an event type.
	// Cleans up both the subscription and stored handler function.
	// No-op if subscriber is nil or not found.
	Unsubscribe(eventType string, subscriber *Subscriber)

	// Publish sends an event to all subscribers of its type.
	// Handlers are called asynchronously in separate goroutines.
	// No-op if event is nil or has no subscribers.
	Publish(event Event)

	PublishData(eventType string, data interface{})
}
