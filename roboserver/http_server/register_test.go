package http_server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"roboserver/database"
	"roboserver/shared"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func init() {
	shared.AppConfig = shared.Config{
		Auth: shared.AuthConfig{
			JWTSecret:   "test-secret-for-http-tests",
			JWTExpiry:   3600,
			NonceLength: 32,
		},
		Database: shared.DatabaseConfig{
			Redis: shared.RedisConfig{
				SessionTTL: "60s",
			},
		},
	}
}

// mockDBManager implements database.DBManager for testing.
type mockDBManager struct {
	pg  *database.PostgresHandler
	rds *database.RedisHandler
}

func (m *mockDBManager) Postgres() *database.PostgresHandler { return m.pg }
func (m *mockDBManager) Redis() *database.RedisHandler       { return m.rds }
func (m *mockDBManager) Stop()                               {}
func (m *mockDBManager) IsHealthy(_ context.Context) bool    { return true }

func newTestServer(db database.DBManager) *HTTPServer_t {
	return &HTTPServer_t{
		db:     db,
		router: chi.NewRouter(),
	}
}

func TestRespondToRegistration_MissingUUID(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	body := strings.NewReader(`{"accept": true}`)
	req := httptest.NewRequest("POST", "/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.respondToRegistration(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rec.Code)
	}
}

func TestRespondToRegistration_InvalidJSON(t *testing.T) {
	s := newTestServer(&mockDBManager{})
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/register", body)
	rec := httptest.NewRecorder()

	s.respondToRegistration(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rec.Code)
	}
}

func TestRespondToRegistration_NilRedis(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	body := strings.NewReader(`{"uuid": "robot-1", "accept": true}`)
	req := httptest.NewRequest("POST", "/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.respondToRegistration(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", rec.Code)
	}
}

func TestGetPendingRegistrations_NilRedis(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	req := httptest.NewRequest("GET", "/register/pending", nil)
	rec := httptest.NewRecorder()

	s.getPendingRegistrations(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", rec.Code)
	}
}

func TestProvisionRobot_MissingFields(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})

	tests := []struct {
		name string
		body string
	}{
		{"missing uuid", `{"public_key": "aabb", "device_type": "test"}`},
		{"missing public_key", `{"uuid": "r1", "device_type": "test"}`},
		{"missing device_type", `{"uuid": "r1", "public_key": "aabb"}`},
		{"all empty", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/provision", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			s.provisionRobot(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for %s, got %d", tc.name, rec.Code)
			}
		})
	}
}

func TestProvisionRobot_InvalidPublicKey(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})

	body := strings.NewReader(`{"uuid": "r1", "public_key": "not-a-key", "device_type": "test"}`)
	req := httptest.NewRequest("POST", "/provision", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.provisionRobot(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid key, got %d", rec.Code)
	}
}

func TestProvisionRobot_NilDatabase(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})

	// Use a valid-length hex key (64 hex chars = 32 bytes Ed25519)
	body := strings.NewReader(`{"uuid": "r1", "public_key": "aabbccdd11223344aabbccdd11223344aabbccdd11223344aabbccdd11223344", "device_type": "test"}`)
	req := httptest.NewRequest("POST", "/provision", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.provisionRobot(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil PG, got %d", rec.Code)
	}
}

func TestGetAllRegisteredRobots_NilDatabase(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	req := httptest.NewRequest("GET", "/provision", nil)
	rec := httptest.NewRecorder()

	s.getAllRegisteredRobots(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil PG, got %d", rec.Code)
	}
}

func TestBlacklistRobot_InvalidJSON(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	body := strings.NewReader(`{bad json}`)
	req := httptest.NewRequest("POST", "/provision/robot-1/blacklist", body)
	req = addChiURLParam(req, "uuid", "robot-1")
	rec := httptest.NewRecorder()

	s.blacklistRobot(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rec.Code)
	}
}

func TestBlacklistRobot_NilDatabase(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	body := strings.NewReader(`{"blacklisted": true}`)
	req := httptest.NewRequest("POST", "/provision/robot-1/blacklist", body)
	req = addChiURLParam(req, "uuid", "robot-1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.blacklistRobot(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", rec.Code)
	}
}

func TestGetRobotStatus_NilRedis(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	req := httptest.NewRequest("GET", "/provision/robot-1/status", nil)
	req = addChiURLParam(req, "uuid", "robot-1")
	rec := httptest.NewRecorder()

	s.getRobotStatus(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil Redis, got %d", rec.Code)
	}
}

func TestGetActiveRobots_NilRedis(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	req := httptest.NewRequest("GET", "/robot", nil)
	rec := httptest.NewRecorder()

	s.getActiveRobots(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil Redis, got %d", rec.Code)
	}
}

func TestGetRobotDetail_NilRedis(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})
	req := httptest.NewRequest("GET", "/robot/robot-1", nil)
	req = addChiURLParam(req, "uuid", "robot-1")
	rec := httptest.NewRecorder()

	s.getRobotDetail(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil Redis, got %d", rec.Code)
	}
}

func TestEphemeralSession_MissingFields(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})

	tests := []struct {
		name string
		body string
	}{
		{"missing uuid", `{"device_type": "test"}`},
		{"missing device_type", `{"uuid": "r1"}`},
		{"all empty", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/ephemeral", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			s.createEphemeralSession(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for %s, got %d", tc.name, rec.Code)
			}
		})
	}
}

func TestEphemeralSession_NilRedis(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})

	body := strings.NewReader(`{"uuid": "r1", "device_type": "test"}`)
	req := httptest.NewRequest("POST", "/ephemeral", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.createEphemeralSession(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil Redis, got %d", rec.Code)
	}
}

func TestDeleteEphemeralSession_NilRedis(t *testing.T) {
	s := newTestServer(&mockDBManager{pg: nil, rds: nil})

	req := httptest.NewRequest("DELETE", "/ephemeral/r1", nil)
	req = addChiURLParam(req, "uuid", "r1")
	rec := httptest.NewRecorder()

	s.deleteEphemeralSession(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", rec.Code)
	}
}

func TestRegistrationResponseSerialization(t *testing.T) {
	req := RegistrationResponse{UUID: "robot-1", Accept: true}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	var decoded RegistrationResponse
	json.Unmarshal(data, &decoded)
	if decoded.UUID != "robot-1" || !decoded.Accept {
		t.Errorf("Serialization mismatch: %+v", decoded)
	}
}

func TestProvisionRequestSerialization(t *testing.T) {
	req := ProvisionRequest{UUID: "r1", PublicKey: "aabb", DeviceType: "test"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	var decoded ProvisionRequest
	json.Unmarshal(data, &decoded)
	if decoded.UUID != "r1" || decoded.PublicKey != "aabb" || decoded.DeviceType != "test" {
		t.Errorf("Serialization mismatch: %+v", decoded)
	}
}

func TestEphemeralRequestSerialization(t *testing.T) {
	req := EphemeralRequest{UUID: "r1", DeviceType: "test", IP: "10.0.0.1"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	var decoded EphemeralRequest
	json.Unmarshal(data, &decoded)
	if decoded.UUID != "r1" || decoded.DeviceType != "test" || decoded.IP != "10.0.0.1" {
		t.Errorf("Serialization mismatch: %+v", decoded)
	}
}

// addChiURLParam injects chi URL params into a request for testing.
func addChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
