package auth

import (
	"roboserver/shared"
	"testing"
	"time"
)

func init() {
	shared.AppConfig = shared.Config{
		Auth: shared.AuthConfig{
			JWTSecret:   "test-secret-key-for-unit-tests",
			JWTExpiry:   3600,
			NonceLength: 32,
		},
	}
}

func TestIssueAndValidateJWT(t *testing.T) {
	token, err := IssueSessionJWT("robot-uuid-123", "proximity_sensor", "192.168.1.100", "sess_abc")
	if err != nil {
		t.Fatalf("Failed to issue JWT: %v", err)
	}
	if token == "" {
		t.Fatal("Token is empty")
	}

	claims, err := ValidateSessionJWT(token)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if claims.Sub != "robot-uuid-123" {
		t.Errorf("Expected sub=robot-uuid-123, got %s", claims.Sub)
	}
	if claims.Type != "proximity_sensor" {
		t.Errorf("Expected type=proximity_sensor, got %s", claims.Type)
	}
	if claims.IP != "192.168.1.100" {
		t.Errorf("Expected ip=192.168.1.100, got %s", claims.IP)
	}
	if claims.SessionID != "sess_abc" {
		t.Errorf("Expected session_id=sess_abc, got %s", claims.SessionID)
	}
}

func TestJWTExpiry(t *testing.T) {
	// Issue a token with 1 second expiry
	origExpiry := shared.AppConfig.Auth.JWTExpiry
	shared.AppConfig.Auth.JWTExpiry = 1
	defer func() { shared.AppConfig.Auth.JWTExpiry = origExpiry }()

	token, err := IssueSessionJWT("robot-1", "test", "127.0.0.1", "sess_1")
	if err != nil {
		t.Fatalf("Failed to issue JWT: %v", err)
	}

	// Should be valid immediately
	_, err = ValidateSessionJWT(token)
	if err != nil {
		t.Fatalf("Token should be valid immediately: %v", err)
	}

	// Wait for expiry
	time.Sleep(2 * time.Second)

	_, err = ValidateSessionJWT(token)
	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got: %v", err)
	}
}

func TestJWTInvalidSignature(t *testing.T) {
	token, _ := IssueSessionJWT("robot-1", "test", "127.0.0.1", "sess_1")
	// Tamper with the token
	tampered := token + "x"
	_, err := ValidateSessionJWT(tampered)
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid for tampered token, got: %v", err)
	}
}

func TestJWTMalformed(t *testing.T) {
	_, err := ValidateSessionJWT("not.a.valid.jwt.string")
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid for malformed token, got: %v", err)
	}

	_, err = ValidateSessionJWT("")
	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid for empty token, got: %v", err)
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := GenerateSessionID()
	id2 := GenerateSessionID()
	if id1 == "" || id2 == "" {
		t.Fatal("Session IDs should not be empty")
	}
	if id1 == id2 {
		t.Error("Session IDs should be unique")
	}
	if len(id1) < 10 {
		t.Errorf("Session ID too short: %s", id1)
	}
}
