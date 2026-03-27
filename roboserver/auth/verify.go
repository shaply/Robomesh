package auth

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
)

var (
	ErrInvalidPublicKey = errors.New("invalid public key")
	ErrInvalidSignature = errors.New("invalid signature")
)

// VerifySignature verifies that signatureHex was produced by signing nonceHex
// with the private key corresponding to publicKeyPEM.
// Supports Ed25519 and ECDSA P-256 keys.
func VerifySignature(publicKeyPEM string, nonceHex string, signatureHex string) error {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return ErrInvalidPublicKey
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return ErrInvalidPublicKey
	}

	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return ErrInvalidSignature
	}

	nonceBytes, err := hex.DecodeString(nonceHex)
	if err != nil {
		return ErrInvalidSignature
	}

	switch key := pub.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(key, nonceBytes, sigBytes) {
			return ErrInvalidSignature
		}
		return nil

	case *ecdsa.PublicKey:
		hash := sha256.Sum256(nonceBytes)
		// ECDSA signature is r || s, each half the curve byte size
		byteLen := (key.Curve.Params().BitSize + 7) / 8
		if len(sigBytes) != 2*byteLen {
			return ErrInvalidSignature
		}
		r := new(big.Int).SetBytes(sigBytes[:byteLen])
		s := new(big.Int).SetBytes(sigBytes[byteLen:])
		if !ecdsa.Verify(key, hash[:], r, s) {
			return ErrInvalidSignature
		}
		return nil

	default:
		return ErrInvalidPublicKey
	}
}

// VerifyEd25519Hex is a convenience function for raw Ed25519 hex-encoded public keys
// (no PEM wrapper), as used by ESP32 and similar embedded devices.
func VerifyEd25519Hex(publicKeyHex string, nonceHex string, signatureHex string) error {
	pubBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}

	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return ErrInvalidSignature
	}

	nonceBytes, err := hex.DecodeString(nonceHex)
	if err != nil {
		return ErrInvalidSignature
	}

	if !ed25519.Verify(ed25519.PublicKey(pubBytes), nonceBytes, sigBytes) {
		return ErrInvalidSignature
	}
	return nil
}

// ParseEd25519PublicKeyHex validates a raw hex-encoded Ed25519 public key.
func ParseEd25519PublicKeyHex(hexKey string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(hexKey)
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}
	return ed25519.PublicKey(b), nil
}

// IsValidPublicKey checks whether the given PEM or hex key can be parsed.
func IsValidPublicKey(key string) bool {
	// Try PEM first
	block, _ := pem.Decode([]byte(key))
	if block != nil {
		_, err := x509.ParsePKIXPublicKey(block.Bytes)
		return err == nil
	}
	// Try raw Ed25519 hex (32 bytes = 64 hex chars)
	if len(key) == 2*ed25519.PublicKeySize {
		_, err := hex.DecodeString(key)
		return err == nil
	}
	// Try raw ECDSA P-256 uncompressed hex (65 bytes = 130 hex chars)
	if len(key) == 130 {
		b, err := hex.DecodeString(key)
		if err != nil {
			return false
		}
		x, y := elliptic.Unmarshal(elliptic.P256(), b)
		return x != nil && y != nil
	}
	return false
}
