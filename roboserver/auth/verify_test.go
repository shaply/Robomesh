package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"testing"
)

func TestVerifyEd25519Hex(t *testing.T) {
	// Generate a test Ed25519 keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	pubHex := hex.EncodeToString(pub)
	nonce := "deadbeefcafebabe0123456789abcdef"
	nonceBytes, _ := hex.DecodeString(nonce)

	sig := ed25519.Sign(priv, nonceBytes)
	sigHex := hex.EncodeToString(sig)

	// Valid signature should pass
	if err := VerifyEd25519Hex(pubHex, nonce, sigHex); err != nil {
		t.Fatalf("Valid signature rejected: %v", err)
	}

	// Tampered signature should fail
	tamperedSig := sigHex[:len(sigHex)-2] + "00"
	if err := VerifyEd25519Hex(pubHex, nonce, tamperedSig); err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature for tampered sig, got: %v", err)
	}

	// Wrong nonce should fail
	if err := VerifyEd25519Hex(pubHex, "ffffffffffffffffffffffffffffffff", sigHex); err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature for wrong nonce, got: %v", err)
	}

	// Invalid public key should fail
	if err := VerifyEd25519Hex("invalid", nonce, sigHex); err != ErrInvalidPublicKey {
		t.Errorf("Expected ErrInvalidPublicKey, got: %v", err)
	}
}

func TestVerifySignaturePEM(t *testing.T) {
	// Generate Ed25519 keypair
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	// Encode as PEM
	pubDER, _ := x509.MarshalPKIXPublicKey(pub)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	nonce := "aabbccdd11223344aabbccdd11223344"
	nonceBytes, _ := hex.DecodeString(nonce)
	sig := ed25519.Sign(priv, nonceBytes)
	sigHex := hex.EncodeToString(sig)

	if err := VerifySignature(string(pubPEM), nonce, sigHex); err != nil {
		t.Fatalf("Valid PEM signature rejected: %v", err)
	}
}

func TestIsValidPublicKey(t *testing.T) {
	// Valid Ed25519 hex key (32 bytes = 64 hex chars)
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	hexKey := hex.EncodeToString(pub)
	if !IsValidPublicKey(hexKey) {
		t.Error("Valid Ed25519 hex key rejected")
	}

	// Valid PEM key
	pubDER, _ := x509.MarshalPKIXPublicKey(pub)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	if !IsValidPublicKey(pemKey) {
		t.Error("Valid PEM key rejected")
	}

	// Invalid key
	if IsValidPublicKey("not-a-key") {
		t.Error("Invalid key accepted")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}
	nonce2, _ := GenerateNonce()

	if nonce1 == nonce2 {
		t.Error("Nonces should be unique")
	}
	// 32 bytes = 64 hex chars
	if len(nonce1) != 64 {
		t.Errorf("Expected nonce length 64, got %d", len(nonce1))
	}
}
