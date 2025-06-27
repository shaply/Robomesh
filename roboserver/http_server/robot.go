package http_server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer) RobotRoutes(r chi.Router) {
	r.Get("/", h.getRobots)
	r.Post("/register", h.registerRobotHandler)
}

func (h *HTTPServer) getRobots(w http.ResponseWriter, r *http.Request) {
	// Handler logic for retrieving robots
	// This would typically involve querying the robot manager
	// and returning a list of registered robots.
	robots := h.robotHandler.GetRobots()
	sendJSONResponse(w, robots, http.StatusOK)
}

func (h *HTTPServer) registerRobotHandler(w http.ResponseWriter, r *http.Request) {
	// Handler logic for registering a robot
	// This would typically involve parsing the request body,
	// validating the data, and storing the robot information.
	sendJSONResponse(w, map[string]string{"message": "Robot registered successfully"}, http.StatusOK)
}
