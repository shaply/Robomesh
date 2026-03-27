package http_server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) RobotRoutes(r chi.Router) {
	r.Get("/", h.getActiveRobots)
	r.Get("/{uuid}", h.getRobotDetail)
}

// getActiveRobots returns all currently active robots from Redis.
func (h *HTTPServer_t) getActiveRobots(w http.ResponseWriter, r *http.Request) {
	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Cache not available", http.StatusServiceUnavailable)
		return
	}

	robots, err := rds.GetAllActiveRobots(r.Context())
	if err != nil {
		http.Error(w, "Failed to get active robots", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robots)
}

// getRobotDetail returns the active session for a specific robot from Redis.
func (h *HTTPServer_t) getRobotDetail(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Cache not available", http.StatusServiceUnavailable)
		return
	}

	active, err := rds.GetActiveRobot(r.Context(), uuid)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"uuid": uuid, "online": false})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"uuid":         uuid,
		"online":       true,
		"ip":           active.IP,
		"device_type":  active.DeviceType,
		"connected_at": active.ConnectedAt,
		"pid":          active.PID,
	})
}
