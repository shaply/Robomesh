package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer) RobotRoutes(r chi.Router) {
	r.Get("/", h.getRobots)
	r.Get("/{robotID}", h.getRobotHandler)                 // Handler to get a specific robot by ID
	r.Get("/{robotID}/quick_action", h.quickActionHandler) // Handler for quick actions on a robot
}

func (h *HTTPServer) getRobots(w http.ResponseWriter, r *http.Request) {
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

func (h *HTTPServer) getRobotHandler(w http.ResponseWriter, r *http.Request) {

	robot := h.validateRobotID(chi.URLParam(r, "robotID"))
	if robot == nil {
		http.Error(w, "Robot not found", http.StatusNotFound)
		return
	}

	sendJSONResponse(w, []byte(robot.ToJSON()), http.StatusOK)
}

// quickActionHandler handles quick actions for a specific robot.
func (h *HTTPServer) quickActionHandler(w http.ResponseWriter, r *http.Request) {
	if h.validateRobotID((chi.URLParam(r, "robotID"))) == nil {
		http.Error(w, "Robot not found", http.StatusNotFound)
		return
	}
	robotHandler, _ := h.rm.GetHandler(chi.URLParam(r, "robotID"), "")
	robotHandler.QuickAction() // Perform the quick action on the robot
	resp := map[string]string{
		"status": "Quick action performed successfully",
		"robot":  robotHandler.GetDeviceID(),
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
	sendJSONResponse(w, respBytes, http.StatusOK)
}

func (h *HTTPServer) validateRobotID(robotID string) shared.Robot {
	if robot, err := h.rm.GetRobot(robotID, ""); err != nil {
		return nil
	} else {
		return robot
	}
}
