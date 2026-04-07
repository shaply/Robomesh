package http_server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"roboserver/handler_engine"
	"testing"
)

func TestListHandlers_Empty(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	req := httptest.NewRequest("GET", "/handler/", nil)
	rec := httptest.NewRecorder()

	s.listHandlers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	var result map[string]int
	json.NewDecoder(rec.Body).Decode(&result)
	if len(result) != 0 {
		t.Errorf("Expected empty handler list, got %d entries", len(result))
	}
}

func TestGetHandlerStatus_NotFound(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	req := httptest.NewRequest("GET", "/handler/nonexistent", nil)
	req = addChiURLParam(req, "uuid", "nonexistent")
	rec := httptest.NewRecorder()

	s.getHandlerStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)
	if result["active"] != false {
		t.Error("Expected active=false for nonexistent handler")
	}
}

func TestKillHandler_NotFound(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	req := httptest.NewRequest("POST", "/handler/nonexistent/kill", nil)
	req = addChiURLParam(req, "uuid", "nonexistent")
	rec := httptest.NewRecorder()

	s.killHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rec.Code)
	}
}

func TestStartHandler_NilDB(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})

	// Ensure not already spawning
	handler_engine.HandlerManager.FinishSpawning("test-uuid")

	req := httptest.NewRequest("POST", "/handler/test-uuid/start", nil)
	req = addChiURLParam(req, "uuid", "test-uuid")
	rec := httptest.NewRecorder()

	s.startHandler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil DB, got %d", rec.Code)
	}
}

func TestStartHandler_AlreadyRunning(t *testing.T) {
	s := newTestServer(&mockDBManager{})

	// Simulate a handler already spawning
	handler_engine.HandlerManager.TryStartSpawning("running-uuid")
	defer handler_engine.HandlerManager.FinishSpawning("running-uuid")

	req := httptest.NewRequest("POST", "/handler/running-uuid/start", nil)
	req = addChiURLParam(req, "uuid", "running-uuid")
	rec := httptest.NewRecorder()

	s.startHandler(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("Expected 409 for already-spawning handler, got %d", rec.Code)
	}
}

func TestStreamHandlerLogs_NoAuth(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	req := httptest.NewRequest("GET", "/handler/nonexistent/logs", nil)
	req = addChiURLParam(req, "uuid", "nonexistent")
	rec := httptest.NewRecorder()

	s.streamHandlerLogs(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}
}
