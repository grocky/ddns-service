package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

const (
	// APIKeyPrefix is prepended to all generated API keys
	APIKeyPrefix = "ddns_sk_"
	// APIKeyBytes is the number of random bytes in an API key
	APIKeyBytes = 32
)

// GenerateAPIKey creates a new secure random API key with the standard prefix.
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, APIKeyBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	return APIKeyPrefix + encoded, nil
}

// HashAPIKey returns the SHA-256 hash of an API key as a hex string.
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ValidateAPIKeyFormat checks if a key has the correct format.
func ValidateAPIKeyFormat(key string) bool {
	if !strings.HasPrefix(key, APIKeyPrefix) {
		return false
	}
	encoded := strings.TrimPrefix(key, APIKeyPrefix)
	if len(encoded) == 0 {
		return false
	}
	// Try to decode to verify it's valid base64url
	_, err := base64.RawURLEncoding.DecodeString(encoded)
	return err == nil
}

// ExtractBearerToken extracts the token from an Authorization header value.
// Returns empty string if the header is not a valid Bearer token.
func ExtractBearerToken(authHeader string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}
	return strings.TrimPrefix(authHeader, prefix)
}

// CompareHashes performs a constant-time comparison of two hash strings.
// Returns true if they match.
func CompareHashes(hash1, hash2 string) bool {
	return subtle.ConstantTimeCompare([]byte(hash1), []byte(hash2)) == 1
}
