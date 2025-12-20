package dns

import (
	"crypto/md5"
	"fmt"
)

const (
	// SubdomainLength is the number of hex characters in the subdomain hash.
	SubdomainLength = 8

	// RootDomain is the base domain for all dynamic DNS subdomains.
	RootDomain = "grocky.net"
)

// GenerateSubdomain creates a deterministic subdomain hash from ownerId and location.
// The hash is the first 8 characters of the MD5 hex digest of "ownerId-location".
func GenerateSubdomain(ownerID, location string) string {
	input := fmt.Sprintf("%s-%s", ownerID, location)
	hash := md5.Sum([]byte(input))
	return fmt.Sprintf("%x", hash)[:SubdomainLength]
}

// FormatFQDN formats a subdomain with the root domain.
func FormatFQDN(subdomain string) string {
	return fmt.Sprintf("%s.%s", subdomain, RootDomain)
}

// BuildACMEChallengeName returns the TXT record name for an ACME challenge.
func BuildACMEChallengeName(subdomain string) string {
	return fmt.Sprintf("_acme-challenge.%s", subdomain)
}
