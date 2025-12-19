package dns

import (
	"testing"

	"gotest.tools/assert"
)

func TestGenerateSubdomain(t *testing.T) {
	testCases := []struct {
		name     string
		ownerID  string
		location string
		expected string
	}{
		{
			name:     "standard input",
			ownerID:  "my-home-lab",
			location: "home",
			expected: "6abf7de6", // md5("my-home-lab-home")[:8]
		},
		{
			name:     "different location",
			ownerID:  "my-home-lab",
			location: "office",
			expected: "71be0bf7", // md5("my-home-lab-office")[:8]
		},
		{
			name:     "different owner",
			ownerID:  "acme-corp",
			location: "office",
			expected: "b7f5de81", // md5("acme-corp-office")[:8]
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GenerateSubdomain(tc.ownerID, tc.location)
			assert.Equal(t, SubdomainLength, len(result))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateSubdomain_Deterministic(t *testing.T) {
	// Same input should always produce same output
	result1 := GenerateSubdomain("test-owner", "home")
	result2 := GenerateSubdomain("test-owner", "home")
	assert.Equal(t, result1, result2)
}

func TestGenerateSubdomain_Unique(t *testing.T) {
	// Different inputs should produce different outputs
	result1 := GenerateSubdomain("owner1", "home")
	result2 := GenerateSubdomain("owner2", "home")
	result3 := GenerateSubdomain("owner1", "office")

	assert.Assert(t, result1 != result2, "different owners should produce different subdomains")
	assert.Assert(t, result1 != result3, "different locations should produce different subdomains")
}

func TestSubdomainLength(t *testing.T) {
	// Verify constant matches expected value
	assert.Equal(t, 8, SubdomainLength)
}

func TestRootDomain(t *testing.T) {
	// Verify root domain is set correctly
	assert.Equal(t, "grocky.net", RootDomain)
}

func TestFormatFQDN(t *testing.T) {
	testCases := []struct {
		name      string
		subdomain string
		expected  string
	}{
		{
			name:      "hash subdomain",
			subdomain: "6abf7de6",
			expected:  "6abf7de6.grocky.net",
		},
		{
			name:      "custom subdomain",
			subdomain: "home",
			expected:  "home.grocky.net",
		},
		{
			name:      "hyphenated subdomain",
			subdomain: "my-home-lab",
			expected:  "my-home-lab.grocky.net",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatFQDN(tc.subdomain)
			assert.Equal(t, tc.expected, result)
		})
	}
}
