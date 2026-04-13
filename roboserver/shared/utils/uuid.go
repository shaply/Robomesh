package utils

import "crypto/rand"

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	// Use crypto/rand for unpredictable IDs — these are used for SSE session
	// identifiers and must not be guessable.
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	for i := range result {
		result[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(result)
}
