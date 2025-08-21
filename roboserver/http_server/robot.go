package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) RobotRoutes(r chi.Router) {
	r.Get("/", h.getRobots)
	r.Get("/robot/{robotID}", h.getRobotHandler)                 // TODO Handler to get a specific robot by ID
	r.Post("/robot/{robotID}", h.postRobotHandler)               // TODO Handler to send information to the robot go routine
	r.Get("/robot/{robotID}/quick_action", h.quickActionHandler) // Handler for quick actions on a robot
	r.Post("/register", h.registerRobotHandler)
}

func (h *HTTPServer_t) getRobots(w http.ResponseWriter, r *http.Request) {
	// Handler logic for retrieving robots
	// This would typically involve querying the robot manager
	// and returning a list of registered robots.
	robots := h.rm.GetRobots()
	jsons := make([]json.RawMessage, 0, len(robots))
	for _, robot := range robots {
		jsons = append(jsons, json.RawMessage(robot.ToJSON()))
	}
	response, err := json.Marshal(jsons)
	if err != nil {
		http.Error(w, "Failed to marshal robots", http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, response, http.StatusOK)
}

func (h *HTTPServer_t) postRobotHandler(w http.ResponseWriter, r *http.Request) {
	robotHandler := h.getRobotHandlerFromIDPath(r)
	if robotHandler == nil {
		http.Error(w, "Robot not found", http.StatusNotFound)
		return
	}
	robotHandler.POST(w, r) // Perform the quick action on the robot
}

func (h *HTTPServer_t) getRobotHandler(w http.ResponseWriter, r *http.Request) {
	robotHandler := h.getRobotHandlerFromIDPath(r)
	if robotHandler == nil {
		http.Error(w, "Robot not found", http.StatusNotFound)
		return
	}
	robotHandler.GET(w, r) // Perform the quick action on the robot
}

// quickActionHandler handles quick actions for a specific robot.
func (h *HTTPServer_t) quickActionHandler(w http.ResponseWriter, r *http.Request) {
	robotHandler := h.getRobotHandlerFromIDPath(r)
	if robotHandler == nil {
		http.Error(w, "Robot not found", http.StatusNotFound)
		return
	}
	robotHandler.QuickAction(w, r) // Perform the quick action on the robot
}

func (h *HTTPServer_t) registerRobotHandler(w http.ResponseWriter, r *http.Request) {
	session := GetSessionFromRequest(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var registerRobotRequest RegisterRobotRequest
	if err := parseJSONRequest(r, &registerRobotRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Handle the registration logic here
	switch registerRobotRequest.Accept {
	case "yes":
		registerRobotRequest.Robot.HandleRegister(h.eb, true)
	case "no":
		registerRobotRequest.Robot.HandleRegister(h.eb, false)
	default:
		http.Error(w, "Invalid accept value", http.StatusBadRequest)
		return
	}
}

func (h *HTTPServer_t) getRobotHandlerFromIDPath(r *http.Request) shared.RobotHandler {
	if h.rm.ValidateRobotID((chi.URLParam(r, "robotID"))) == nil {
		return nil
	}
	robotHandler, err := h.rm.GetHandler(chi.URLParam(r, "robotID"), "")
	if err != nil {
		return nil
	}
	return robotHandler
}
