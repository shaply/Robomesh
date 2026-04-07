package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/handler_engine"

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

// getRobotDetail returns a comprehensive view of a robot including active session,
// heartbeat state, handler status, and registration info.
func (h *HTTPServer_t) getRobotDetail(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Cache not available", http.StatusServiceUnavailable)
		return
	}

	resp := map[string]interface{}{
		"uuid":   uuid,
		"online": false,
	}

	// Active session info
	if active, err := rds.GetActiveRobot(r.Context(), uuid); err == nil {
		resp["online"] = true
		resp["ip"] = active.IP
		resp["device_type"] = active.DeviceType
		resp["connected_at"] = active.ConnectedAt
		resp["pid"] = active.PID
	}

	// Heartbeat info (independent of handler)
	if hb, err := rds.GetHeartbeat(r.Context(), uuid); err == nil {
		resp["heartbeat"] = map[string]interface{}{
			"last_seq":  hb.LastSeq,
			"last_seen": hb.LastSeen,
			"ip":        hb.IP,
		}
	}

	// Handler status
	if hp, ok := handler_engine.HandlerManager.Get(uuid); ok {
		resp["handler"] = map[string]interface{}{
			"active":      true,
			"pid":         hp.PID,
			"device_type": hp.DeviceType,
		}
	} else {
		resp["handler"] = map[string]interface{}{
			"active": false,
		}
	}

	// Registration info from PostgreSQL
	if pg := h.db.Postgres(); pg != nil {
		if robot, err := pg.GetRobotByUUID(r.Context(), uuid); err == nil {
			resp["registered"] = true
			resp["registration"] = map[string]interface{}{
				"device_type":    robot.DeviceType,
				"is_blacklisted": robot.IsBlacklisted,
				"created_at":     robot.CreatedAt,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
