package http_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"roboserver/auth"
	"roboserver/handler_engine"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) HandlerRoutes(r chi.Router) {
	r.Get("/", h.listHandlers)
	r.Get("/types", h.listHandlerTypes)
	r.Get("/{uuid}", h.getHandlerStatus)
	r.Post("/{uuid}/start", h.startHandler)
	r.Post("/{uuid}/kill", h.killHandler)
	// Log streaming moved to semi-public route with ticket-based auth (see http_server.go)
}

// listHandlerTypes returns all available handler types (device types with handler scripts).
func (h *HTTPServer_t) listHandlerTypes(w http.ResponseWriter, r *http.Request) {
	types := handler_engine.ListHandlerTypes()
	if types == nil {
		types = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

// listHandlers returns all currently running handler processes.
func (h *HTTPServer_t) listHandlers(w http.ResponseWriter, r *http.Request) {
	handlers := handler_engine.HandlerManager.ListAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handlers)
}

// getHandlerStatus checks if a handler is running for a specific robot.
func (h *HTTPServer_t) getHandlerStatus(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	hp, ok := handler_engine.HandlerManager.Get(uuid)

	resp := map[string]interface{}{
		"uuid":   uuid,
		"active": ok,
	}
	if ok {
		resp["pid"] = hp.PID
		resp["device_type"] = hp.DeviceType
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// startHandler spawns a handler process for a robot that doesn't currently have one.
func (h *HTTPServer_t) startHandler(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")

	// Atomically check and mark as spawning to prevent concurrent spawn races
	if !handler_engine.HandlerManager.TryStartSpawning(uuid) {
		http.Error(w, "Handler already running or being started", http.StatusConflict)
		return
	}
	defer handler_engine.HandlerManager.FinishSpawning(uuid)

	pg := h.db.Postgres()
	rds := h.db.Redis()
	if pg == nil || rds == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Look up robot info — try active session first, then PostgreSQL
	var deviceType, ip string
	if active, _ := rds.GetActiveRobot(r.Context(), uuid); active != nil {
		deviceType = active.DeviceType
		ip = active.IP
	} else if robot, err := pg.GetRobotByUUID(r.Context(), uuid); err == nil {
		deviceType = robot.DeviceType
		// IP unknown since robot isn't connected; check heartbeat
		if hb, _ := rds.GetHeartbeat(r.Context(), uuid); hb != nil {
			ip = hb.IP
		}
	} else {
		http.Error(w, "Robot not found", http.StatusNotFound)
		return
	}

	sessionID := "manual-" + auth.GenerateSessionID()

	// Use the server's long-lived context (not r.Context()) so the handler
	// survives after the HTTP response is sent.
	hp, err := handler_engine.SpawnHandlerProcess(
		h.ctx,
		uuid, deviceType, ip, sessionID,
		pg, rds, h.bus,
		nil, // No direct robot TCP connection
	)
	if err != nil {
		shared.DebugPrint("Failed to start handler for %s: %v", uuid, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "started",
		"uuid":   uuid,
		"pid":    hp.PID,
	})
}

// killHandler stops a running handler process.
func (h *HTTPServer_t) killHandler(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")

	if err := handler_engine.HandlerManager.Kill(uuid); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "killed",
		"uuid":   uuid,
	})
}

// streamHandlerLogs opens an SSE stream of handler stdout/stderr log lines.
// Accepts ticket-based auth (?ticket=...) or JWT from Authorization header/cookie,
// since browser EventSource cannot send custom headers.
func (h *HTTPServer_t) streamHandlerLogs(w http.ResponseWriter, r *http.Request) {
	session := h.validateTicket(r)
	if session == nil {
		session = h.validateSessionFull(r)
	}
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	uuid := chi.URLParam(r, "uuid")

	if !handler_engine.HandlerManager.Has(uuid) {
		http.Error(w, "No handler running for this robot", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Channel log events to serialize writes to the ResponseWriter
	logCh := make(chan []byte, 256)

	// Subscribe to handler log events
	topic := fmt.Sprintf("handler.%s.log", uuid)
	cancel, err := h.bus.SubscribeEvent(topic, func(eventType string, data any) {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return
		}
		select {
		case logCh <- jsonData:
		default:
			// Drop if channel is full to avoid blocking event bus
		}
	})
	if err != nil {
		http.Error(w, "Failed to subscribe to logs", http.StatusInternalServerError)
		return
	}
	defer cancel()

	fmt.Fprintf(w, "data: {\"uuid\":%q,\"line\":\"Connected to log stream\",\"stream\":\"system\"}\n\n", uuid)
	flusher.Flush()

	// Drain channel, writing to the ResponseWriter from this single goroutine
	for {
		select {
		case <-r.Context().Done():
			return
		case data := <-logCh:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
