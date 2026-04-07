package http_server

import (
	"net/http"
	"net/http/httptest"
	"roboserver/shared"
	"strings"
	"testing"
)

func TestLoginHandler_InvalidJSON(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/auth/login", body)
	rec := httptest.NewRecorder()

	s.loginHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rec.Code)
	}
}

func TestLoginHandler_NilRedis(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	body := strings.NewReader(`{"username": "admin", "password": "pass"}`)
	req := httptest.NewRequest("POST", "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.loginHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", rec.Code)
	}
}

func TestLogoutHandler_NoToken(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	rec := httptest.NewRecorder()

	s.logoutHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}
}

func TestCheckToken_NoToken(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	req := httptest.NewRequest("GET", "/auth", nil)
	rec := httptest.NewRecorder()

	s.checkToken(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}
}

func TestCheckToken_InvalidToken(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	req := httptest.NewRequest("GET", "/auth", nil)
	req.Header.Set("Authorization", "Bearer invalid-jwt-token")
	rec := httptest.NewRecorder()

	s.checkToken(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}
}

func TestIssueTicket_NoAuth(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	req := httptest.NewRequest("POST", "/auth/ticket", nil)
	rec := httptest.NewRecorder()

	s.issueTicketHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}
}

func TestExtractRawToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		cookie   string
		expected string
	}{
		{"bearer header", "Bearer mytoken123", "", "mytoken123"},
		{"raw header", "mytoken123", "", "mytoken123"},
		{"cookie", "", "mytoken123", "mytoken123"},
		{"empty", "", "", ""},
		{"bearer takes precedence", "Bearer headertoken", "cookietoken", "headertoken"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "session-token", Value: tc.cookie})
			}
			result := extractRawToken(req)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestValidateSession(t *testing.T) {
	if err := ValidateSession(nil); err == nil {
		t.Error("Expected error for nil session")
	}

	sess := &shared.Session{UserID: "admin", SessionID: "sess-1"}
	if err := ValidateSession(sess); err != nil {
		t.Errorf("Expected nil error for valid session, got %v", err)
	}
}

func TestLoginRateLimiter(t *testing.T) {
	ip := "test-rate-limit-ip"

	// Should not be rate limited initially
	if checkLoginRate(ip) {
		t.Error("Should not be rate limited initially")
	}

	// Record max attempts
	for i := 0; i < loginMaxAttempts; i++ {
		recordLoginAttempt(ip)
	}

	// Should now be rate limited
	if !checkLoginRate(ip) {
		t.Error("Should be rate limited after max attempts")
	}

	// Different IP should not be rate limited
	if checkLoginRate("different-ip") {
		t.Error("Different IP should not be rate limited")
	}
}
