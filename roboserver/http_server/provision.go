package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/auth"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) ProvisionRoutes(r chi.Router) {
	r.Get("/", h.getAllRegisteredRobots)
	r.Post("/", h.provisionRobot)
	r.Get("/{uuid}", h.getRobotRecord)
	r.Post("/{uuid}/blacklist", h.blacklistRobot)
	r.Get("/{uuid}/status", h.getRobotStatus)
}

type ProvisionRequest struct {
	UUID       string `json:"uuid"`
	PublicKey  string `json:"public_key"`
	DeviceType string `json:"device_type"`
}

// provisionRobot registers a new robot's public key in PostgreSQL.
func (h *HTTPServer_t) provisionRobot(w http.ResponseWriter, r *http.Request) {
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UUID == "" || req.PublicKey == "" || req.DeviceType == "" {
		http.Error(w, "uuid, public_key, and device_type are required", http.StatusBadRequest)
		return
	}

	if !auth.IsValidPublicKey(req.PublicKey) {
		http.Error(w, "Invalid public key format", http.StatusBadRequest)
		return
	}

	pg := h.db.Postgres()
	if pg == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	if err := pg.RegisterRobot(r.Context(), req.UUID, req.PublicKey, req.DeviceType); err != nil {
		shared.DebugPrint("Failed to provision robot: %v", err)
		http.Error(w, "Failed to provision robot", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "provisioned", "uuid": req.UUID})
}

// getRobotRecord returns the PostgreSQL record for a robot.
func (h *HTTPServer_t) getRobotRecord(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	pg := h.db.Postgres()
	if pg == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	robot, err := pg.GetRobotByUUID(r.Context(), uuid)
	if err != nil {
		http.Error(w, "Robot not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robot)
}

// blacklistRobot toggles the blacklist flag on a robot.
func (h *HTTPServer_t) blacklistRobot(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	var req struct {
		Blacklisted bool `json:"blacklisted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pg := h.db.Postgres()
	if pg == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	if err := pg.BlacklistRobot(r.Context(), uuid, req.Blacklisted); err != nil {
		http.Error(w, "Failed to update blacklist", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"uuid": uuid, "blacklisted": req.Blacklisted})
}

// getAllRegisteredRobots returns all robots from the PostgreSQL registry.
func (h *HTTPServer_t) getAllRegisteredRobots(w http.ResponseWriter, r *http.Request) {
	pg := h.db.Postgres()
	if pg == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	robots, err := pg.GetAllRobots(r.Context())
	if err != nil {
		shared.DebugPrint("Failed to get registered robots: %v", err)
		http.Error(w, "Failed to get robots", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robots)
}

// getRobotStatus checks Redis for the robot's active session state.
func (h *HTTPServer_t) getRobotStatus(w http.ResponseWriter, r *http.Request) {
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
	})
}
