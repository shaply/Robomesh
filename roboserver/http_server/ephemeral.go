package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/auth"
	"roboserver/database"
	"roboserver/shared"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) EphemeralRoutes(r chi.Router) {
	r.Post("/", h.createEphemeralSession)
	r.Delete("/{uuid}", h.deleteEphemeralSession)
}

type EphemeralRequest struct {
	UUID       string `json:"uuid"`
	DeviceType string `json:"device_type"`
	IP         string `json:"ip"`
}

// createEphemeralSession creates a temporary robot session in Redis only.
// No PostgreSQL record or public key verification is required.
func (h *HTTPServer_t) createEphemeralSession(w http.ResponseWriter, r *http.Request) {
	var req EphemeralRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UUID == "" || req.DeviceType == "" {
		http.Error(w, "uuid and device_type are required", http.StatusBadRequest)
		return
	}

	if req.IP == "" {
		req.IP = r.RemoteAddr
	}

	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Cache not available", http.StatusServiceUnavailable)
		return
	}

	// Check if UUID already has an active session
	if existing, _ := rds.GetActiveRobot(r.Context(), req.UUID); existing != nil {
		http.Error(w, "UUID already has an active session", http.StatusConflict)
		return
	}

	// Check if UUID is already registered in PostgreSQL
	if pg := h.db.Postgres(); pg != nil {
		if robot, _ := pg.GetRobotByUUID(r.Context(), req.UUID); robot != nil {
			http.Error(w, "UUID belongs to a registered robot", http.StatusConflict)
			return
		}
	}

	sessionID := auth.GenerateSessionID()
	jwt, err := auth.IssueSessionJWT(req.UUID, req.DeviceType, req.IP, sessionID)
	if err != nil {
		shared.DebugPrint("Failed to issue ephemeral JWT: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	ttl := shared.AppConfig.Database.Redis.TTL()
	active := &database.ActiveRobot{
		UUID:        req.UUID,
		IP:          req.IP,
		DeviceType:  req.DeviceType,
		SessionJWT:  jwt,
		ConnectedAt: time.Now().Unix(),
	}

	if err := rds.SetActiveRobot(r.Context(), active, ttl); err != nil {
		http.Error(w, "Failed to store session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"uuid":       req.UUID,
		"session_id": sessionID,
		"token":      jwt,
		"ttl":        ttl.Seconds(),
	})
}

// deleteEphemeralSession removes an ephemeral robot session from Redis.
func (h *HTTPServer_t) deleteEphemeralSession(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Cache not available", http.StatusServiceUnavailable)
		return
	}

	if err := rds.RemoveActiveRobot(r.Context(), uuid); err != nil {
		http.Error(w, "Failed to remove session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "removed", "uuid": uuid})
}
