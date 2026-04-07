package http_server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeartbeat_NilDB(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})

	// DB check happens before JSON parsing, so any body returns 503
	body := strings.NewReader(`{"uuid": "robot-1", "payload": "{}", "signature": "aabb"}`)
	req := httptest.NewRequest("POST", "/heartbeat", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleHeartbeat(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil DB, got %d", rec.Code)
	}
}

func TestHeartbeat_InvalidJSON_NilDB(t *testing.T) {
	// With nil DB, returns 503 regardless of body
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/heartbeat", body)
	rec := httptest.NewRecorder()

	s.handleHeartbeat(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil DB, got %d", rec.Code)
	}
}
