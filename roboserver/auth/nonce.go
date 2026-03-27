package auth

import (
	"crypto/rand"
	"encoding/hex"
	"roboserver/shared"
)

// GenerateNonce creates a cryptographically random hex-encoded nonce.
func GenerateNonce() (string, error) {
	b := make([]byte, shared.AppConfig.Auth.NonceLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
