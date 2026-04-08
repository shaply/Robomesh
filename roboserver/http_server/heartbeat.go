package http_server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"roboserver/auth"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) HeartbeatRoutes(r chi.Router) {
	r.Post("/", h.handleHeartbeat)
}

// handleHeartbeat processes an HTTP heartbeat from a robot.
// Request body: { "uuid": "...", "payload": "...", "signature": "..." }
// The payload is the JSON string that was signed, and signature is hex-encoded.
func (h *HTTPServer_t) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	pg := h.db.Postgres()
	rds := h.db.Redis()
	if pg == nil || rds == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		UUID      string `json:"uuid"`
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Extract IP from request
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	result, err := auth.ProcessHeartbeat(r.Context(), req.UUID, req.Payload, req.Signature, ip, pg, rds)
	if err != nil {
		shared.DebugPrint("HTTP heartbeat failed for %s: %v", req.UUID, err)
		http.Error(w, "Heartbeat rejected", http.StatusUnauthorized)
		return
	}

	// Publish heartbeat event
	if h.bus != nil {
		h.bus.PublishEvent(fmt.Sprintf("robot.%s.heartbeat", result.UUID), result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
