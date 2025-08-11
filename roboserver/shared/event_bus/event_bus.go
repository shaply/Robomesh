package event_bus

import (
	"roboserver/shared"
	"roboserver/shared/data_structures"
)

var limiter = make(chan bool, shared.EVENT_BUS_BUFFER_SIZE) // Channel to limit event bus publishing rate

func NewEventBus() EventBus {
	return &EventBus_t{
		subscriptions: data_structures.NewSafeMap[string, *data_structures.SafeSet[Subscriber]](),
		handlers:      data_structures.NewSafeMap[Subscriber, *data_structures.SafeMap[string, SubscriberHandler]](),
	}
}

func (eb *EventBus_t) Subscribe(eventType string, subscriber *Subscriber, handler SubscriberHandler) *Subscriber {
	if subscriber == nil || eventType == "" {
		subscriber = NewSubscriber()
	}

	// Store the handler function
	eb.handlers.GetOrDefault(*subscriber, data_structures.NewSafeMap[string, SubscriberHandler]()).Set(eventType, handler)

	// Ensure the handler was actually set (retry logic)
	for {
		handlers := eb.handlers.GetOrDefault(*subscriber, data_structures.NewSafeMap[string, SubscriberHandler]())

		if _, exists := handlers.Get(eventType); exists {
			break // Handler successfully set
		}

		// Handler not found, try setting again
		handlers.Set(eventType, handler)
	}

	// Add subscriber to set
	eb.subscriptions.GetOrDefault(eventType, data_structures.NewSafeSet[Subscriber]()).Add(*subscriber)

	// Ensure the subscriber is stored
	for {
		subscribers := eb.subscriptions.GetOrDefault(eventType, data_structures.NewSafeSet[Subscriber]())
		if exists := subscribers.Contains(*subscriber); exists {
			break // Subscriber successfully added
		}
		subscribers.Add(*subscriber) // Retry adding subscriber
	}

	return subscriber
}

func (eb *EventBus_t) Unsubscribe(eventType string, subscriber *Subscriber) {
	if subscriber == nil {
		return
	}

	if eventType == "" {
		// Unsubscribe from all events
		events, ok := eb.handlers.Get(*subscriber)
		if !ok {
			return
		}
		for _, event := range events.GetKeys() {
			eb.Unsubscribe(event, subscriber)
		}
		return
	}

	// Remove subscriber from multiset
	if multiset, ok := eb.subscriptions.Get(eventType); ok {
		multiset.Remove(*subscriber)
		eb.subscriptions.DeleteIfEmpty(eventType)
	}
	if handlers, ok := eb.handlers.Get(*subscriber); ok {
		handlers.Delete(eventType)
		eb.handlers.DeleteIfEmpty(*subscriber)
	}
}

func (eb *EventBus_t) Publish(event Event) {
	if event == nil || event.GetType() == "" {
		return
	}

	eventType := event.GetType()

	shared.DebugPrint("Publishing event: %s", eventType)

	if subscribers, ok := eb.subscriptions.Get(eventType); ok {
		ch := subscribers.Iterate()
		for sub := range ch {
			if mp, ok := eb.handlers.Get(sub); ok {
				if handler, ok := mp.Get(eventType); ok {
					limiter <- true // Limit the number of concurrent handlers
					go func() {
						defer func() { <-limiter }()
						handler(event)
					}()
				} else {
					go eb.Unsubscribe(eventType, &sub) // Unsubscribe if handler not found
				}
			} else {
				go eb.Unsubscribe(eventType, &sub) // Unsubscribe if subscriber not found
			}
		}
	}
}

func (eb *EventBus_t) PublishData(eventType string, data interface{}) {
	if eventType == "" {
		shared.DebugPrint("Cannot publish event with empty type")
		return
	}

	if data == nil {
		shared.DebugPrint("Cannot publish event %s with nil data", eventType)
		return
	}

	event := NewDefaultEvent(eventType, data)
	eb.Publish(event)
}
