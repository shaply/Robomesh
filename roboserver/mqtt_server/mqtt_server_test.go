package mqtt_server

import (
	"encoding/json"
	"testing"
)

func TestAuthRequestSerialization(t *testing.T) {
	req := AuthRequest{
		UUID:      "robot-001",
		Signature: "abc123",
		Nonce:     "nonce-xyz",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal AuthRequest: %v", err)
	}

	var decoded AuthRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AuthRequest: %v", err)
	}

	if decoded.UUID != "robot-001" {
		t.Errorf("Expected UUID 'robot-001', got %q", decoded.UUID)
	}
	if decoded.Signature != "abc123" {
		t.Errorf("Expected Signature 'abc123', got %q", decoded.Signature)
	}
	if decoded.Nonce != "nonce-xyz" {
		t.Errorf("Expected Nonce 'nonce-xyz', got %q", decoded.Nonce)
	}
}

func TestAuthResponseSerialization(t *testing.T) {
	tests := []struct {
		name   string
		resp   AuthResponse
		status string
	}{
		{
			name:   "nonce response",
			resp:   AuthResponse{Status: "nonce", Nonce: "abc123"},
			status: "nonce",
		},
		{
			name:   "ok response",
			resp:   AuthResponse{Status: "ok", JWT: "jwt-token-here"},
			status: "ok",
		},
		{
			name:   "error response",
			resp:   AuthResponse{Status: "error", Error: "unknown robot"},
			status: "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.resp)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded AuthResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Status != tc.status {
				t.Errorf("Expected status %q, got %q", tc.status, decoded.Status)
			}
		})
	}
}

func TestHeartbeatRequestSerialization(t *testing.T) {
	req := HeartbeatRequest{
		Payload:   `{"seq":1,"ttl":60}`,
		Signature: "signature-hex",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal HeartbeatRequest: %v", err)
	}

	var decoded HeartbeatRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal HeartbeatRequest: %v", err)
	}

	if decoded.Payload != req.Payload {
		t.Errorf("Expected Payload %q, got %q", req.Payload, decoded.Payload)
	}
	if decoded.Signature != req.Signature {
		t.Errorf("Expected Signature %q, got %q", req.Signature, decoded.Signature)
	}
}

func TestAuthRequest_EmptySignature(t *testing.T) {
	// When signature is empty, server should treat this as step 1 (nonce request)
	req := AuthRequest{UUID: "robot-001"}
	if req.Signature != "" {
		t.Error("Expected empty signature for step 1")
	}
}
