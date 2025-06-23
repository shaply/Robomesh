package http_server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer) RobotRoutes(r chi.Router) {
	r.Post("/register", h.registerRobotHandler)
}

func (h *HTTPServer) registerRobotHandler(w http.ResponseWriter, r *http.Request) {
	// Handler logic for registering a robot
	// This would typically involve parsing the request body,
	// validating the data, and storing the robot information.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Robot registered successfully"))
}
