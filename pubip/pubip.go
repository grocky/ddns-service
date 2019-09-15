package pubip

//IP function to get the public ip address
func IP() (string, error) {
	client := newAuthority("https://icanhazip.com/")
	ipAddress, err := client.requestExternalIP()
	return ipAddress, err
}
