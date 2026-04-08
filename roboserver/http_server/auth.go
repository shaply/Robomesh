package http_server

import (
	"encoding/json"
	"net"
	"net/http"
	"roboserver/auth"
	"roboserver/shared"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

// loginRateLimiter tracks login attempts per IP.
var loginRateLimiter = struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}{
	attempts: make(map[string][]time.Time),
}

func init() {
	go cleanupRateLimiter()
}

// cleanupRateLimiter periodically evicts stale entries from the rate limiter map.
func cleanupRateLimiter() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		loginRateLimiter.mu.Lock()
		cutoff := time.Now().Add(-loginWindow)
		for ip, attempts := range loginRateLimiter.attempts {
			valid := attempts[:0]
			for _, t := range attempts {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(loginRateLimiter.attempts, ip)
			} else {
				loginRateLimiter.attempts[ip] = valid
			}
		}
		loginRateLimiter.mu.Unlock()
	}
}

const (
	loginMaxAttempts = 5
	loginWindow      = 5 * time.Minute
)

// checkLoginRate returns true if the IP has exceeded the login rate limit.
func checkLoginRate(ip string) bool {
	loginRateLimiter.mu.Lock()
	defer loginRateLimiter.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-loginWindow)

	// Filter out expired attempts
	attempts := loginRateLimiter.attempts[ip]
	valid := attempts[:0]
	for _, t := range attempts {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	loginRateLimiter.attempts[ip] = valid

	return len(valid) >= loginMaxAttempts
}

func recordLoginAttempt(ip string) {
	loginRateLimiter.mu.Lock()
	defer loginRateLimiter.mu.Unlock()
	loginRateLimiter.attempts[ip] = append(loginRateLimiter.attempts[ip], time.Now())
}

func (h *HTTPServer_t) AuthRoutes(r chi.Router) {
	r.Get("/", h.checkToken)
	r.Post("/login", h.loginHandler)
	r.Post("/logout", h.logoutHandler)
	// Ticket endpoint requires valid JWT (header/cookie) — returns a short-lived single-use ticket for SSE
	r.Post("/ticket", h.issueTicketHandler)

	// Protected: password change (requires valid session)
	r.Group(func(r chi.Router) {
		r.Use(h.SessionValidationMiddleware)
		r.Post("/password", h.changePasswordHandler)
	})
}

func (h *HTTPServer_t) checkToken(w http.ResponseWriter, r *http.Request) {
	session := h.validateSessionFull(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPServer_t) loginHandler(w http.ResponseWriter, r *http.Request) {
	// Rate limit by IP
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	if checkLoginRate(ip) {
		http.Error(w, "Too many login attempts. Try again later.", http.StatusTooManyRequests)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Reject excessively long passwords before bcrypt to prevent CPU DoS.
	// bcrypt truncates at 72 bytes anyway, so anything longer is pointless.
	if len(loginReq.Password) > 72 {
		recordLoginAttempt(ip)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Validate credentials against Redis
	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	user, err := rds.GetUser(r.Context(), loginReq.Username)
	if err != nil {
		recordLoginAttempt(ip)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginReq.Password)); err != nil {
		recordLoginAttempt(ip)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Issue JWT
	token, err := auth.IssueUserJWT(loginReq.Username)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Store session in Redis for server-side invalidation
	ttl := shared.AppConfig.Database.Redis.UserTTL()
	if err := rds.SetUserSession(r.Context(), token, loginReq.Username, ttl); err != nil {
		shared.DebugPrint("Failed to store user session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Logged in successfully",
		"token":   token,
	}

	shared.DebugPrint("AUTH: User %s logged in", loginReq.Username)

	responseBytes, _ := json.Marshal(response)
	sendJSONResponse(w, responseBytes, http.StatusOK)
}

func (h *HTTPServer_t) logoutHandler(w http.ResponseWriter, r *http.Request) {
	token := extractRawToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Remove session from Redis
	rds := h.db.Redis()
	if rds != nil {
		rds.RemoveUserSession(r.Context(), token)
	}

	sendJSONResponse(w, []byte(`{"status": "success", "message": "Logged out successfully"}`), http.StatusOK)
}

// validateSessionFull validates JWT and checks that the session still exists in Redis.
// This prevents use of tokens after logout.
func (h *HTTPServer_t) validateSessionFull(r *http.Request) *shared.Session {
	token := extractRawToken(r)
	if token == "" {
		return nil
	}
	session := parseSessionFromToken(token)
	if session == nil {
		return nil
	}
	// Verify session still exists in Redis
	if rds := h.db.Redis(); rds != nil {
		username, err := rds.GetUserSession(r.Context(), token)
		if err != nil || username != session.UserID {
			return nil
		}
	}
	return session
}

// extractRawToken pulls the raw JWT string from the request.
// Only checks Authorization header and cookie — NOT query params (tokens in URLs are a security risk).
func extractRawToken(r *http.Request) string {
	// Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
		return authHeader
	}

	// Cookie fallback
	if cookie, err := r.Cookie("session-token"); err == nil {
		return cookie.Value
	}

	return ""
}

// parseSessionFromToken validates a JWT and returns a Session.
func parseSessionFromToken(token string) *shared.Session {
	claims, err := auth.ValidateUserJWT(token)
	if err != nil {
		return nil
	}

	return &shared.Session{
		UserID:    claims.Sub,
		SessionID: claims.TokenID,
	}
}

func ValidateSession(session *shared.Session) error {
	if session == nil {
		return shared.ErrUnauthorized
	}
	return nil
}

const ticketTTL = 30 * time.Second

// issueTicketHandler creates a short-lived single-use ticket for SSE connections.
// Requires a valid JWT in the Authorization header or cookie (NOT query param).
func (h *HTTPServer_t) issueTicketHandler(w http.ResponseWriter, r *http.Request) {
	session := h.validateSessionFull(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	ticket, err := auth.GenerateNonce()
	if err != nil {
		http.Error(w, "Failed to generate ticket", http.StatusInternalServerError)
		return
	}

	if err := rds.SetTicket(r.Context(), ticket, session.UserID, ticketTTL); err != nil {
		http.Error(w, "Failed to store ticket", http.StatusInternalServerError)
		return
	}

	sendResponseAsJSON(w, map[string]string{"ticket": ticket}, http.StatusOK)
}

// changePasswordHandler allows authenticated users to change their password.
func (h *HTTPServer_t) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	session := h.validateSessionFull(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) > 72 {
		http.Error(w, "Password must not exceed 72 characters", http.StatusBadRequest)
		return
	}

	rds := h.db.Redis()
	if rds == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Verify current password
	user, err := rds.GetUser(r.Context(), session.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password and store
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	user.PasswordHash = string(newHash)
	if err := rds.SetUser(r.Context(), user); err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	shared.DebugPrint("AUTH: User %s changed password", session.UserID)
	sendJSONResponse(w, []byte(`{"status":"success","message":"Password changed successfully"}`), http.StatusOK)
}

// validateTicket consumes a single-use ticket and returns a session.
func (h *HTTPServer_t) validateTicket(r *http.Request) *shared.Session {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		return nil
	}
	rds := h.db.Redis()
	if rds == nil {
		return nil
	}
	username, err := rds.ConsumeTicket(r.Context(), ticket)
	if err != nil || username == "" {
		return nil
	}
	return &shared.Session{
		UserID:    username,
		SessionID: "ticket",
	}
}
