package pubip

// IP returns the public IP address of the current machine
// by querying an external authority service.
func IP() (string, error) {
	client := newAuthority("https://icanhazip.com/")
	return client.requestExternalIP()
}
