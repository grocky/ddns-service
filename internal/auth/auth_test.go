package auth

import (
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	assert.NilError(t, err)
	assert.Assert(t, strings.HasPrefix(key, APIKeyPrefix), "expected key to have prefix %s, got %s", APIKeyPrefix, key)

	// Key should be unique each time
	key2, err := GenerateAPIKey()
	assert.NilError(t, err)
	assert.Assert(t, key != key2, "expected unique keys, got same key twice")
}

func TestGenerateAPIKey_Format(t *testing.T) {
	key, err := GenerateAPIKey()
	assert.NilError(t, err)

	// Remove prefix and verify base64url encoding
	encoded := strings.TrimPrefix(key, APIKeyPrefix)
	assert.Assert(t, len(encoded) > 0, "expected non-empty encoded part")

	// Validate the key format
	assert.Assert(t, ValidateAPIKeyFormat(key), "generated key should pass validation")
}

func TestHashAPIKey(t *testing.T) {
	key := "ddns_sk_testkey123"
	hash := HashAPIKey(key)

	// Hash should be 64 hex characters (SHA-256 = 32 bytes = 64 hex chars)
	assert.Equal(t, 64, len(hash))

	// Same key should produce same hash
	hash2 := HashAPIKey(key)
	assert.Equal(t, hash, hash2)

	// Different keys should produce different hashes
	hash3 := HashAPIKey("ddns_sk_differentkey")
	assert.Assert(t, hash != hash3, "different keys should have different hashes")
}

func TestHashAPIKey_Consistency(t *testing.T) {
	// Test that hashing is deterministic
	testCases := []struct {
		key          string
		expectedHash string
	}{
		{
			key:          "ddns_sk_abc123",
			expectedHash: HashAPIKey("ddns_sk_abc123"),
		},
		{
			key:          "",
			expectedHash: HashAPIKey(""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			hash := HashAPIKey(tc.key)
			assert.Equal(t, tc.expectedHash, hash)
		})
	}
}

func TestValidateAPIKeyFormat(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "valid generated key",
			key:      "ddns_sk_7Kx9mP2qR5vW8yB3nF6hJ4tL1cA0eD9gXXXXXXXXXXXX",
			expected: true,
		},
		{
			name:     "valid base64url characters",
			key:      "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
			expected: true,
		},
		{
			name:     "valid with hyphens and underscores",
			key:      "ddns_sk_abc-def_ghi",
			expected: true,
		},
		{
			name:     "missing prefix",
			key:      "7Kx9mP2qR5vW8yB3nF6hJ4tL1cA0eD9g",
			expected: false,
		},
		{
			name:     "wrong prefix",
			key:      "api_key_7Kx9mP2qR5vW8yB3nF6hJ4tL1cA0eD9g",
			expected: false,
		},
		{
			name:     "prefix only",
			key:      "ddns_sk_",
			expected: false,
		},
		{
			name:     "empty string",
			key:      "",
			expected: false,
		},
		{
			name:     "invalid base64 characters",
			key:      "ddns_sk_invalid!@#$%",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateAPIKeyFormat(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractBearerToken(t *testing.T) {
	testCases := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer ddns_sk_testtoken123",
			expected:   "ddns_sk_testtoken123",
		},
		{
			name:       "bearer token with spaces in value",
			authHeader: "Bearer token with spaces",
			expected:   "token with spaces",
		},
		{
			name:       "missing bearer prefix",
			authHeader: "ddns_sk_testtoken123",
			expected:   "",
		},
		{
			name:       "lowercase bearer",
			authHeader: "bearer ddns_sk_testtoken123",
			expected:   "",
		},
		{
			name:       "basic auth",
			authHeader: "Basic dXNlcjpwYXNz",
			expected:   "",
		},
		{
			name:       "empty header",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "bearer with no token",
			authHeader: "Bearer ",
			expected:   "",
		},
		{
			name:       "bearer only",
			authHeader: "Bearer",
			expected:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractBearerToken(tc.authHeader)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCompareHashes(t *testing.T) {
	hash1 := HashAPIKey("ddns_sk_testkey1")
	hash2 := HashAPIKey("ddns_sk_testkey1")
	hash3 := HashAPIKey("ddns_sk_testkey2")

	// Same hashes should match
	assert.Assert(t, CompareHashes(hash1, hash2), "same hashes should match")

	// Different hashes should not match
	assert.Assert(t, !CompareHashes(hash1, hash3), "different hashes should not match")

	// Empty strings
	assert.Assert(t, CompareHashes("", ""), "empty strings should match")
	assert.Assert(t, !CompareHashes(hash1, ""), "hash and empty should not match")
}

func TestCompareHashes_ConstantTime(t *testing.T) {
	// This test verifies the function works correctly
	// Actual timing analysis would require more sophisticated testing
	hash := HashAPIKey("ddns_sk_secret")
	wrongHash := HashAPIKey("ddns_sk_wrong")

	// Both should complete and return correct results
	assert.Assert(t, CompareHashes(hash, hash))
	assert.Assert(t, !CompareHashes(hash, wrongHash))
}
