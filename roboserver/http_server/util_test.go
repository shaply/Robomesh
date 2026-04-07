package http_server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"roboserver/shared"
	"strings"
	"testing"
)

func TestSendResponseAsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	data := map[string]string{"key": "value"}
	sendResponseAsJSON(rec, data, http.StatusOK)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected application/json, got %s", ct)
	}

	var result map[string]string
	json.NewDecoder(rec.Body).Decode(&result)
	if result["key"] != "value" {
		t.Errorf("Expected value, got %s", result["key"])
	}
}

func TestSendJSONResponse_ValidJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	sendJSONResponse(rec, []byte(`{"ok":true}`), http.StatusOK)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}
}

func TestSendJSONResponse_InvalidJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	sendJSONResponse(rec, []byte(`not json`), http.StatusOK)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for invalid JSON, got %d", rec.Code)
	}
}

func TestParseJSONRequest(t *testing.T) {
	body := strings.NewReader(`{"name": "test"}`)
	req := httptest.NewRequest("POST", "/", body)

	var result struct {
		Name string `json:"name"`
	}
	err := parseJSONRequest(req, &result)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Expected 'test', got %s", result.Name)
	}
}

func TestParseJSONRequest_Invalid(t *testing.T) {
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/", body)

	var result struct{}
	err := parseJSONRequest(req, &result)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestSessionValidationMiddleware_NoAuth(t *testing.T) {
	s := newTestServer(&mockDBManager{})

	handler := s.SessionValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", rec.Code)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	shared.AppConfig.Server.AllowedOrigins = []string{"http://localhost:5173"}
	s := newTestServer(&mockDBManager{})

	handler := s.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 for OPTIONS, got %d", rec.Code)
	}
	if acao := rec.Header().Get("Access-Control-Allow-Origin"); acao != "http://localhost:5173" {
		t.Errorf("Expected allowed origin, got %s", acao)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	shared.AppConfig.Server.AllowedOrigins = []string{"http://localhost:5173"}
	s := newTestServer(&mockDBManager{})

	handler := s.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if acao := rec.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Errorf("Expected no ACAO header for disallowed origin, got %s", acao)
	}
}
