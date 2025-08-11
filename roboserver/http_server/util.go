package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/shared"
)

func sendResponseAsJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		// Note: Can't call http.Error here since headers are already written
		// Just log the error or handle it appropriately for your use case
		shared.DebugErrorf("Error encoding JSON response: %v", err)
		return
	}
}

func sendJSONResponse(w http.ResponseWriter, data_json []byte, status int) {
	if !json.Valid(data_json) {
		var err error
		data_json, err = json.Marshal(data_json)
		if err != nil {
			http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(data_json); err != nil {
		http.Error(w, "Error writing JSON response", http.StatusInternalServerError)
	}
}

func parseJSONRequest(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}
	return nil
}
