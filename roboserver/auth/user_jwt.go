package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"roboserver/shared"
	"strings"
	"time"
)

type UserJWTClaims struct {
	Sub      string `json:"sub"`      // Username
	Iat      int64  `json:"iat"`      // Issued at
	Exp      int64  `json:"exp"`      // Expiry
	TokenID  string `json:"token_id"` // Unique token identifier
}

// IssueUserJWT creates a signed JWT for a user session.
func IssueUserJWT(username string) (string, error) {
	secret := shared.AppConfig.Auth.JWTSecret
	if secret == "" {
		return "", ErrTokenInvalid
	}

	tokenID := make([]byte, 16)
	if _, err := rand.Read(tokenID); err != nil {
		return "", fmt.Errorf("failed to generate token ID: %w", err)
	}

	now := time.Now().Unix()
	claims := UserJWTClaims{
		Sub:     username,
		Iat:     now,
		Exp:     now + int64(shared.AppConfig.Auth.JWTExpiry),
		TokenID: hex.EncodeToString(tokenID),
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
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig, nil
}

// ValidateUserJWT parses and validates a user JWT, returning the claims.
func ValidateUserJWT(tokenStr string) (*UserJWTClaims, error) {
	secret := shared.AppConfig.Auth.JWTSecret
	if secret == "" {
		return nil, ErrTokenInvalid
	}

	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expectedSig)) != 1 {
		return nil, ErrTokenInvalid
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrTokenInvalid
	}

	var claims UserJWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrTokenInvalid
	}

	if time.Now().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}
