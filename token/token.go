package token

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
)

const TokenBytes = 32

// Generate creates a cryptographically random hex token.
func Generate() (string, error) {
	b := make([]byte, TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Validate compares a submitted token against the stored token in constant time.
// Returns true if they match.
func Validate(submitted, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(submitted), []byte(stored)) == 1
}
