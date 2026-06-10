package token

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
)

const KeyBytes = 32

// Generate creates a cryptographically random base64url-encoded key.
func Generate() (string, error) {
	b := make([]byte, KeyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Validate compares a submitted token against the stored token in constant time.
// Returns true if they match.
func Validate(submitted, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(submitted), []byte(stored)) == 1
}
