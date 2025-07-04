package http_server

import (
	"encoding/json"
	"net/http"
)

func sendResponseAsJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
	} else {
		w.WriteHeader(status)
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
