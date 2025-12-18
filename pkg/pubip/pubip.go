package pubip

// IPVersion specifies which IP protocol version to use.
type IPVersion int

const (
	// IPv4 requests an IPv4 address.
	IPv4 IPVersion = iota
	// IPv6 requests an IPv6 address.
	IPv6
)

var authorityURLs = map[IPVersion]string{
	IPv4: "https://ipv4.icanhazip.com/",
	IPv6: "https://ipv6.icanhazip.com/",
}

// IP returns the public IP address of the current machine
// by querying an external authority service.
// The version parameter specifies whether to return IPv4 or IPv6.
func IP(version IPVersion) (string, error) {
	url, ok := authorityURLs[version]
	if !ok {
		url = authorityURLs[IPv4]
	}
	client := newAuthority(url)
	return client.requestExternalIP()
}
