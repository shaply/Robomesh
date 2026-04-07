package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"roboserver/shared"
	"strings"
	"time"
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("invalid token")
)

type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type JWTClaims struct {
	Sub       string `json:"sub"`        // Robot UUID
	Type      string `json:"type"`       // Device type
	IP        string `json:"ip"`         // Connected IP
	Iat       int64  `json:"iat"`        // Issued at
	Exp       int64  `json:"exp"`        // Expiry
	SessionID string `json:"session_id"` // Unique session identifier
}

// IssueSessionJWT creates a signed JWT for a verified robot session.
func IssueSessionJWT(uuid, deviceType, ip, sessionID string) (string, error) {
	secret := shared.AppConfig.Auth.JWTSecret
	if secret == "" {
		return "", errors.New("JWT_SECRET not configured")
	}

	now := time.Now().Unix()
	claims := JWTClaims{
		Sub:       uuid,
		Type:      deviceType,
		IP:        ip,
		Iat:       now,
		Exp:       now + int64(shared.AppConfig.Auth.JWTExpiry),
		SessionID: sessionID,
	}

	header := JWTHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT claims: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	sig := signHS256(signingInput, secret)

	return signingInput + "." + sig, nil
}

// ValidateSessionJWT parses and validates a JWT, returning the claims.
func ValidateSessionJWT(tokenStr string) (*JWTClaims, error) {
	secret := shared.AppConfig.Auth.JWTSecret
	if secret == "" {
		return nil, errors.New("JWT_SECRET not configured")
	}

	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := signHS256(signingInput, secret)
	if parts[2] != expectedSig {
		return nil, ErrTokenInvalid
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrTokenInvalid
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrTokenInvalid
	}

	if time.Now().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

func signHS256(input, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// GenerateSessionID creates a unique session identifier.
func GenerateSessionID() string {
	nonce, err := GenerateNonce()
	if err != nil {
		// Fallback: this should never happen as GenerateNonce uses crypto/rand
		return fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("sess_%s", nonce[:16])
}
