package event_bus

import (
	"roboserver/shared"
	"roboserver/shared/data_structures"
	"sync/atomic"
)

// inFlight counts concurrent handler goroutines. Publishers drop events
// rather than block once we hit EVENT_BUS_BUFFER_SIZE, so a slow/stuck
// subscriber can never stall the publish path (and therefore the server's
// network goroutines that call into it).
var inFlight atomic.Int64

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

	// Store the handler function — GetOrDefault returns the existing or newly-inserted map,
	// then we set the handler on it. No retry loop needed: if a concurrent Unsubscribe
	// removes the entry, re-subscribing is the caller's responsibility.
	eb.handlers.GetOrDefault(*subscriber, data_structures.NewSafeMap[string, SubscriberHandler]()).Set(eventType, handler)

	// Add subscriber to set
	eb.subscriptions.GetOrDefault(eventType, data_structures.NewSafeSet[Subscriber]()).Add(*subscriber)

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
		for _, sub := range subscribers.Snapshot() {
			if mp, ok := eb.handlers.Get(sub); ok {
				if handler, ok := mp.Get(eventType); ok {
					// Non-blocking backpressure: drop rather than stall the publisher
					// (which is usually a network goroutine).
					if inFlight.Load() >= int64(shared.EVENT_BUS_BUFFER_SIZE) {
						shared.DebugPrint("Event bus saturated, dropping event: %s", eventType)
						continue
					}
					inFlight.Add(1)
					go func() {
						defer func() {
							inFlight.Add(-1)
							if r := recover(); r != nil {
								shared.DebugPrint("Event handler panic on %s: %v", eventType, r)
							}
						}()
						handler(event)
					}()
				} else {
					subCopy := sub
					go eb.Unsubscribe(eventType, &subCopy) // Unsubscribe if handler not found
				}
			} else {
				subCopy := sub
				go eb.Unsubscribe(eventType, &subCopy) // Unsubscribe if subscriber not found
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
