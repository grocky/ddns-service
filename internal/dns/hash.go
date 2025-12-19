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

// FullSubdomain returns the complete subdomain including the root domain.
// Example: "a3f8c2d1.grocky.net"
func FullSubdomain(ownerID, location string) string {
	return fmt.Sprintf("%s.%s", GenerateSubdomain(ownerID, location), RootDomain)
}
