package http_server

import (
	"encoding/json"
	"net/http"
	"roboserver/shared"
)

// maxRequestBodySize limits JSON request bodies to 1 MB to prevent memory
// exhaustion from oversized payloads. Individual endpoints can override this
// by calling http.MaxBytesReader directly before decoding.
const maxRequestBodySize = 1 << 20 // 1 MB

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
		http.Error(w, "Invalid JSON response data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data_json)
}

func parseJSONRequest(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}
	return nil
}
