package http_server

import (
	"fmt"
	"net/http"
	"roboserver/http_server/http_events"
	"roboserver/shared"
	"strings"
)

// eventsHandler handles SSE connections. Accepts either a single-use ticket (?ticket=...)
// or a JWT from Authorization header/cookie. Tickets are preferred for browser EventSource
// since it cannot set custom headers.
func (h *HTTPServer_t) eventsHandler(w http.ResponseWriter, r *http.Request) {
	// Try ticket first (for EventSource connections), then JWT
	session := h.validateTicket(r)
	if session == nil {
		session = h.validateSessionFull(r)
	}
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get event names from query parameter
	// URL example: /events?events=robot_status,door_open,sensor_data
	eventNames := []string{}
	if eventsParam := r.URL.Query().Get("events"); eventsParam != "" {
		// Split comma-separated event names
		eventNames = strings.Split(eventsParam, ",")
		// Trim whitespace from each event name
		for i, name := range eventNames {
			eventNames[i] = strings.TrimSpace(name)
		}
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	// Note: Access-Control-Allow-Origin is handled by global CORS middleware

	// Send initial retry directive
	fmt.Fprintf(w, "retry: 3000\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	eSess := http_events.NewEventSession(session)

	// Create a session validator that periodically checks Redis for session validity.
	// This closes the SSE connection if the user logs out or the session is revoked.
	token := extractRawToken(r)
	validator := func() bool {
		if token == "" {
			return false
		}
		rds := h.db.Redis()
		if rds == nil {
			return false
		}
		username, err := rds.GetUserSession(r.Context(), token)
		return err == nil && username == session.UserID
	}

	client := h.sseManager.RegisterClient(eSess, w, validator)

	shared.DebugPrint("Registered new SSE client (user=%s) subscribed to %v", eSess.Session.UserID, eventNames)

	// Subscribe to specific events if provided
	if len(eventNames) > 0 {
		for _, eventName := range eventNames {
			client.SubscribeToEvent(eventName)
		}
	}

	<-r.Context().Done()
	h.sseManager.UnregisterClient(eSess)
}

func (h *HTTPServer_t) eventsSubscribeHandler(w http.ResponseWriter, r *http.Request) {
	sess := h.validateSessionFull(r)
	if sess == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var eStruct http_events.EventStruct
	if err := parseJSONRequest(r, &eStruct); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	client, ok := h.sseManager.GetClient(&eStruct.ESess)
	if !ok {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	for _, eventType := range eStruct.EventTypes {
		if eventType == "" {
			continue
		}
		client.SubscribeToEvent(eventType)
	}

	sendResponseAsJSON(w, map[string]interface{}{"status": "subscribed", "events": eStruct.EventTypes}, http.StatusOK)
}

func (h *HTTPServer_t) eventsUnsubscribeHandler(w http.ResponseWriter, r *http.Request) {
	sess := h.validateSessionFull(r)
	if sess == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var eStruct http_events.EventStruct
	if err := parseJSONRequest(r, &eStruct); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	client, ok := h.sseManager.GetClient(&eStruct.ESess)
	if !ok {
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	for _, eventType := range eStruct.EventTypes {
		if eventType == "" {
			continue
		}
		client.UnsubscribeFromEvent(eventType)
	}

	shared.DebugPrint("Client %v unsubscribed from events %v", client, eStruct.EventTypes)
	sendResponseAsJSON(w, map[string]interface{}{"status": "unsubscribed", "events": eStruct.EventTypes}, http.StatusOK)
}
