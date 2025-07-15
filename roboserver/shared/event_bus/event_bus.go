package event_bus

import "roboserver/shared/event_bus/data_structures"

func NewEventBus() EventBus {
	return &EventBus_t{
		subscriptions: data_structures.NewSafeMap[string, *data_structures.Set[Subscriber]](),
		handlers:      data_structures.NewSafeMap[Subscriber, SubscriberHandler](),
	}
}

func (eb *EventBus_t) Subscribe(eventType string, subscriber *Subscriber, handler SubscriberHandler) *Subscriber {
	if subscriber == nil {
		subscriber = NewSubscriber()
	}

	// Store the handler function
	eb.handlers.Set(*subscriber, handler)

	// Add subscriber to set
	set := eb.subscriptions.GetOrDefault(eventType, data_structures.NewSet[Subscriber]())
	set.Add(*subscriber)
	eb.subscriptions.Set(eventType, set)
	return subscriber
}

func (eb *EventBus_t) Unsubscribe(eventType string, subscriber *Subscriber) {
	if subscriber == nil {
		return
	}

	// Remove subscriber from multiset
	if multiset, ok := eb.subscriptions.Get(eventType); ok {
		multiset.Remove(*subscriber)
	}

	// Remove handler function
	eb.handlers.Delete(*subscriber)
}

func (eb *EventBus_t) Publish(event Event) {
	if event == nil {
		return
	}

	eventType := event.GetType()
	if subscribers, ok := eb.subscriptions.Get(eventType); ok {
		ch := subscribers.Iterate()
		for sub := range ch {
			if handler, ok := eb.handlers.Get(sub); ok {
				go handler(event)
			}
		}
	}
}
