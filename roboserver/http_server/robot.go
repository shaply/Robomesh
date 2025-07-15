package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

func (h *HTTPServer_t) RobotRoutes(r chi.Router) {
	r.Get("/", h.getRobots)
	r.Get("/ws", h.wsHandler)
	r.Get("/robot/{robotID}", h.getRobotHandler)                 // Handler to get a specific robot by ID
	r.Get("/robot/{robotID}/quick_action", h.quickActionHandler) // Handler for quick actions on a robot
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

func (h *HTTPServer_t) getRobotHandler(w http.ResponseWriter, r *http.Request) {

	robot := h.validateRobotID(chi.URLParam(r, "robotID"))
	if robot == nil {
		http.Error(w, "Robot not found", http.StatusNotFound)
		return
	}

	sendJSONResponse(w, []byte(robot.ToJSON()), http.StatusOK)
}

// quickActionHandler handles quick actions for a specific robot.
func (h *HTTPServer_t) quickActionHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *HTTPServer_t) validateRobotID(robotID string) shared.Robot {
	if robot, err := h.rm.GetRobot(robotID, ""); err != nil {
		return nil
	} else {
		return robot
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebSocket connections, TODO: Implement proper origin checks
	},
}

// TODO: Implement WebSocket handling logic
func (h *HTTPServer_t) wsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		shared.DebugPrint("Failed to upgrade connection:", err)
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

}
