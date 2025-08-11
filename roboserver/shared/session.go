package shared

// Session represents a user session.
type Session struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}
