package pubip

import (
	"context"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestQueryWithConsensus_TwoAgree(t *testing.T) {
	// Mock the authorities to return controlled responses
	originalAuthorities := authorities
	defer func() { authorities = originalAuthorities }()

	// We can't easily mock HTTP in the consensus test without more refactoring,
	// so we'll test the real function with actual URLs in integration tests.
	// For unit tests, we test the core logic directly.

	ctx := context.Background()

	// Test with URLs that should all return the same IP
	// This is effectively an integration test
	urls := []string{
		"https://ipv4.icanhazip.com/",
		"https://checkip.amazonaws.com/",
	}

	ip, err := queryWithConsensus(ctx, urls)
	assert.NilError(t, err)
	assert.Assert(t, ip != "", "expected non-empty IP")
	assert.Assert(t, isValidIPv4(ip), "expected valid IPv4, got: %s", ip)
}

func TestQueryWithConsensus_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Use URLs that will timeout
	urls := []string{
		"http://10.255.255.1/", // Non-routable IP, will timeout
	}

	_, err := queryWithConsensus(ctx, urls)
	assert.Assert(t, err != nil, "expected error due to timeout")
}

func TestIP_IPv4(t *testing.T) {
	ip, err := IP(IPv4)
	assert.NilError(t, err)
	assert.Assert(t, ip != "", "expected non-empty IP")
	assert.Assert(t, isValidIPv4(ip), "expected valid IPv4, got: %s", ip)
}

// isValidIPv4 checks if a string looks like a valid IPv4 address.
func isValidIPv4(ip string) bool {
	parts := 0
	for _, c := range ip {
		if c == '.' {
			parts++
		} else if c < '0' || c > '9' {
			return false
		}
	}
	return parts == 3
}
