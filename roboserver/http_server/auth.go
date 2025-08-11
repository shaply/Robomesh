package http_server

// Implement use of JWTs for session management
// Implement with redis later to quick blacklist or invalidate JWTs

// Methods right now are just for demonstration purposes

import (
	"encoding/json"
	"net/http"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) AuthRoutes(r chi.Router) {
	r.Get("/", h.checkToken) // Endpoint to check if the token is valid
	r.Post("/login", h.loginHandler)
	r.Post("/logout", h.logoutHandler) // Endpoint to log out and invalidate the session
}

func (h *HTTPServer_t) checkToken(w http.ResponseWriter, r *http.Request) {
	session := GetSessionFromRequest(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// If we reach here, the token is valid
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPServer_t) loginHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Parse login credentials from request
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// 2. Validate credentials (check against database/store)
	userID, err := h.validateCredentials(loginReq.Username, loginReq.Password)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// 3. Create a session token (JWT or session ID)
	sessionToken, err := h.createSessionToken(userID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// 4. Send the token back to client (JSON response for cross-origin)
	// Note: Cookies don't work reliably for cross-origin requests
	response := map[string]interface{}{
		"status":  "success",
		"message": "Logged in successfully",
		"token":   sessionToken,
	}

	shared.DebugPrint("AUTH: Created session token '%s' for user %s", sessionToken, userID)

	responseBytes, _ := json.Marshal(response)
	sendJSONResponse(w, responseBytes, http.StatusOK)
}

func (h *HTTPServer_t) logoutHandler(w http.ResponseWriter, r *http.Request) {
	session := GetSessionFromRequest(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	sendJSONResponse(w, []byte(`{"status": "success", "message": "Logged out successfully"}`), http.StatusOK)
}

// Helper method to validate user credentials
func (h *HTTPServer_t) validateCredentials(username, password string) (string, error) {
	// TODO: Implement actual credential validation
	// This should check against your user database/store

	// Placeholder implementation
	if username == "admin" && password == "password" {
		return "user-123", nil // Return user ID
	}

	return "", shared.ErrUnauthorized
}

// Helper method to create session token (JWT or similar)
func (h *HTTPServer_t) createSessionToken(userID string) (string, error) {
	// TODO: Implement JWT token creation
	// For now, return a simple token
	return "jwt-token-" + userID, nil
}

// GetSessionFromRequest extracts session from Authorization header or cookie
func GetSessionFromRequest(r *http.Request) *shared.Session {
	// First, try Authorization header (for cross-origin requests)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Support both "Bearer token" and just "token" formats
		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
		return parseSessionFromToken(token)
	}

	// Fallback to cookie (for same-origin requests)
	if cookie, err := r.Cookie("session-token"); err == nil {
		return parseSessionFromToken(cookie.Value)
	}

	// Fallback to auth-token GET parameter
	if token := r.URL.Query().Get("auth-token"); token != "" {
		return parseSessionFromToken(token) // might fail bc URI encoded
	}

	shared.DebugPrint("AUTH: No session found in Authorization header or cookies")
	return nil
}

// Helper to parse session from token
func parseSessionFromToken(token string) *shared.Session {
	// TODO: Implement JWT parsing or session lookup
	// For now, return a mock session for valid tokens
	if token != "" {
		return &shared.Session{
			UserID:    "user-123",
			SessionID: token,
		}
	}
	return nil
}

func ValidateSession(session *shared.Session) error {
	if session == nil {
		return shared.ErrUnauthorized
	}
	return nil
}
