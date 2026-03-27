package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) RegisterRoutes(r chi.Router) {
	r.Post("/", h.respondToRegistration)
	r.Get("/pending", h.getPendingRegistrations)
}

type RegistrationResponse struct {
	UUID   string `json:"uuid"`
	Accept bool   `json:"accept"`
}

// respondToRegistration handles accept/reject of a pending robot registration.
// This is called by the frontend notification or terminal.
func (h *HTTPServer_t) respondToRegistration(w http.ResponseWriter, r *http.Request) {
	var req RegistrationResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UUID == "" {
		http.Error(w, "uuid is required", http.StatusBadRequest)
		return
	}

	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Cache not available", http.StatusServiceUnavailable)
		return
	}

	// Verify the pending registration exists
	_, err := rds.GetPendingRobot(r.Context(), req.UUID)
	if err != nil {
		http.Error(w, "No pending registration found for this UUID", http.StatusNotFound)
		return
	}

	// Publish accept/reject via comms bus (TCP server is waiting on this)
	if err := h.bus.PublishRegistrationResponse(r.Context(), req.UUID, req.Accept); err != nil {
		shared.DebugPrint("Failed to publish registration response for %s: %v", req.UUID, err)
		http.Error(w, "Failed to send response", http.StatusInternalServerError)
		return
	}

	action := "rejected"
	if req.Accept {
		action = "accepted"
	}

	shared.DebugPrint("Robot %s registration %s via HTTP", req.UUID, action)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"uuid":   req.UUID,
		"status": action,
	})
}

// getPendingRegistrations returns all robots awaiting approval.
func (h *HTTPServer_t) getPendingRegistrations(w http.ResponseWriter, r *http.Request) {
	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Cache not available", http.StatusServiceUnavailable)
		return
	}

	pending, err := rds.GetAllPendingRobots(r.Context())
	if err != nil {
		http.Error(w, "Failed to get pending registrations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pending)
}
