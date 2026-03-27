package http_events

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"roboserver/comms"
	"roboserver/shared"
	"roboserver/shared/data_structures"
	"roboserver/shared/utils"
	"sync"
	"sync/atomic"
)

// TODO:
// It seems that the Event isn't getting sent, so either not properly subscribed or not properly handled
// Implement the session flow between the HTTP server and the WebSocket server

type EventsClient struct {
	Writer  http.ResponseWriter
	Session EventSession
	manager *EventsManager_t
	done    chan struct{}

	// cancelFuncs tracks subscription cancellation functions by event type.
	cancelFuncs map[string]func()
	cancelMu    sync.Mutex

	msgQueue *data_structures.SafeQueue[*comms.Event] // Queue for outgoing messages
	ended    atomic.Bool                               // Indicates if the client has ended
}

func NewEventsClient(sess *EventSession, w http.ResponseWriter, manager *EventsManager_t) *EventsClient {
	return &EventsClient{
		Writer:      w,
		Session:     *sess,
		manager:     manager,
		done:        make(chan struct{}),
		cancelFuncs: make(map[string]func()),
		msgQueue:    data_structures.NewSafeQueue[*comms.Event](true),
		ended:       atomic.Bool{},
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

	// Cancel all event subscriptions
	client.cancelMu.Lock()
	for _, cancel := range client.cancelFuncs {
		cancel()
	}
	client.cancelFuncs = nil
	client.cancelMu.Unlock()
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
		client.sendSSEEvent(event.Type, event.Data, fmt.Sprintf("%d", eventID))
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

	cancel, err := client.manager.bus.SubscribeEvent(eventType, func(et string, data any) {
		if client.ended.Load() {
			return
		}
		client.msgQueue.Enqueue(&comms.Event{Type: et, Data: data})
	})
	if err != nil {
		shared.DebugError(fmt.Errorf("failed to subscribe to event %s: %v", eventType, err))
		return
	}

	client.cancelMu.Lock()
	// Cancel existing subscription for this event type if any
	if existing, ok := client.cancelFuncs[eventType]; ok {
		existing()
	}
	client.cancelFuncs[eventType] = cancel
	client.cancelMu.Unlock()
}

func (client *EventsClient) UnsubscribeFromEvent(eventType string) {
	if client.ended.Load() {
		shared.DebugError(fmt.Errorf("client has ended, cannot unsubscribe from event %s",
			eventType))
		return
	}

	client.cancelMu.Lock()
	if cancel, ok := client.cancelFuncs[eventType]; ok {
		cancel()
		delete(client.cancelFuncs, eventType)
	}
	client.cancelMu.Unlock()
}
