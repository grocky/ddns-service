package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grocky/ddns-service/pkg/pubip"
)

var ipv6 = flag.Bool("6", false, "return IPv6 address instead of IPv4")

func main() {
	flag.Parse()
	setupProfiling()
	defer stopProfiling()

	version := pubip.IPv4
	if *ipv6 {
		version = pubip.IPv6
	}

	ip, err := pubip.IP(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println(ip)
}
