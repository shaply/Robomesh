package http_events

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"roboserver/shared"
	"roboserver/shared/data_structures"
	"roboserver/shared/event_bus"
	"roboserver/shared/utils"
	"sync/atomic"
)

// TODO:
// It seems that the Event isn't getting sent, so either not properly subscribed or not properly handled
// Implement the session flow between the HTTP server and the WebSocket server

type EventsClient struct {
	Writer     http.ResponseWriter
	Session    EventSession
	Subscriber *event_bus.Subscriber
	manager    *EventsManager_t
	done       chan struct{}
	msgQueue   *data_structures.SafeQueue[event_bus.Event] // Queue for outgoing messages

	ended atomic.Bool // Indicates if the client has ended
}

func NewEventsClient(sess *EventSession, w http.ResponseWriter, manager *EventsManager_t) *EventsClient {
	return &EventsClient{
		Writer:     w,
		Session:    *sess,
		Subscriber: event_bus.NewSubscriber(),
		manager:    manager,
		done:       make(chan struct{}),
		msgQueue:   data_structures.NewSafeQueue[event_bus.Event](true),
		ended:      atomic.Bool{},
	}
}

func (client *EventsClient) Start() {
	client.ended.Store(false)

	// TODO: Add session validation logic go routine
	go client.ReadMsgQueue()
}

func (client *EventsClient) cleanup() {
	if client.ended.Load() {
		shared.DebugError(fmt.Errorf("client already ended, cannot cleanup again"))
		return
	}
	client.ended.Store(true)
	utils.SafeCloseChannel(client.done)
	utils.SafeClose(client.msgQueue)
	client.manager.clients.Delete(client.Session)
	client.manager.eb.Unsubscribe("", client.Subscriber) // Unsubscribe from all events
}

func (client *EventsClient) ReadMsgQueue() {
	defer client.cleanup()

	eventID := 0

	// Send initial connection confirmation event
	client.sendSSEEvent(EVENT_TYPE_SESSION_ID, client.Session, fmt.Sprintf("%d", eventID))

	for !client.ended.Load() {
		event, ok := client.msgQueue.Read(true, client.done)
		if !ok {
			return
		}

		// Check for nil event to prevent panic
		if event == nil {
			shared.DebugError(fmt.Errorf("received nil event from queue for client %v", client.Session))
			continue
		}

		eventID++
		client.sendSSEEvent(event.GetType(), event.GetData(), fmt.Sprintf("%d", eventID))
	}
}

// sendSSEEvent sends a properly formatted SSE event with optional event ID
func (client *EventsClient) sendSSEEvent(eventType string, data interface{}, id string) {
	// Check if client has ended before sending
	if client.ended.Load() {
		shared.DebugError(fmt.Errorf("client %v has ended, cannot send SSE event %s", client.Session, eventType))
		return
	}

	// Convert to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		shared.DebugError(fmt.Errorf("failed to marshal event: %v", err))
		return
	}

	// Base64 encode to make it completely safe for SSE
	encodedEventData := base64.StdEncoding.EncodeToString(jsonData)

	var eventStruct SentEvent
	eventStruct.Id = id
	eventStruct.Type = eventType
	eventStruct.EncodedData = encodedEventData

	jsonData, err = json.Marshal(eventStruct)
	if err != nil {
		shared.DebugError(fmt.Errorf("failed to marshal event struct: %v", err))
		return
	}
	encodedData := base64.StdEncoding.EncodeToString(jsonData)
	fmt.Fprintf(client.Writer, "data: %s\n\n", encodedData)

	// Flush immediately
	if flusher, ok := client.Writer.(http.Flusher); ok {
		flusher.Flush()
	} else {
		shared.DebugError(fmt.Errorf("client %v Writer does not support flushing", client.Session))
	}
}

func (client *EventsClient) SubscribeToEvent(eventType string) {
	if client.ended.Load() {
		shared.DebugError(fmt.Errorf("client has ended, cannot subscribe to event %s",
			eventType))
		return
	}

	client.manager.eb.Subscribe(eventType, client.Subscriber, client.HandleEvent)
}

func (client *EventsClient) UnsubscribeFromEvent(eventType string) {
	if client.ended.Load() {
		shared.DebugError(fmt.Errorf("client has ended, cannot unsubscribe from event %s",
			eventType))
		return
	}
	client.manager.eb.Unsubscribe(eventType, client.Subscriber)
}

func (client *EventsClient) HandleEvent(event event_bus.Event) {
	if client.ended.Load() {
		shared.DebugError(fmt.Errorf("client has ended, cannot handle event %s",
			event.GetType()))
		return
	}
	client.msgQueue.Enqueue(event)
}
