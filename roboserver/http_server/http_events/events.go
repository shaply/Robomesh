package http_events

import (
	"net/http"
	"roboserver/shared/data_structures"
	"roboserver/shared/event_bus"
)

type EventsManager_t struct {
	eb      event_bus.EventBus
	clients *data_structures.SafeMap[EventSession, *EventsClient]
}

func NewEventsManager(eb event_bus.EventBus) *EventsManager_t {
	return &EventsManager_t{
		eb:      eb,
		clients: data_structures.NewSafeMap[EventSession, *EventsClient](),
	}
}

// RegisterClient registers a new WebSocket client with the EventsManager.
func (em *EventsManager_t) RegisterClient(sess *EventSession, w http.ResponseWriter) *EventsClient {
	client := NewEventsClient(sess, w, em)
	oldClient, exists := em.clients.Pop(*sess)
	if exists {
		oldClient.cleanup() // Clean up old client resources
	}
	em.clients.Set(*sess, client)
	client.Start()
	return client
}

func (em *EventsManager_t) UnregisterClient(sess *EventSession) {
	client, exists := em.clients.Pop(*sess)
	if !exists {
		return
	}

	client.cleanup() // Clean up the client resources
}

func (em *EventsManager_t) GetClient(sess *EventSession) (*EventsClient, bool) {
	client, exists := em.clients.Get(*sess)
	if !exists || client.ended.Load() {
		return nil, false
	}
	return client, true
}
