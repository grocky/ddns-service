package pubip

import (
	"context"
	"errors"
	"sync"
	"time"
)

// IPVersion specifies which IP protocol version to use.
type IPVersion int

const (
	// IPv4 requests an IPv4 address.
	IPv4 IPVersion = iota
	// IPv6 requests an IPv6 address.
	IPv6
)

// consensusRequired is the number of authorities that must agree.
const consensusRequired = 2

// authorities lists the IP lookup services for each protocol version.
var authorities = map[IPVersion][]string{
	IPv4: {
		"https://ipv4.icanhazip.com/",
		"https://checkip.amazonaws.com/",
		"https://api.ipify.org/",
		"https://ifconfig.me/ip",
	},
	IPv6: {
		"https://ipv6.icanhazip.com/",
		"https://api6.ipify.org/",
		"https://ifconfig.co/ip",
	},
}

// result holds the response from an authority query.
type result struct {
	ip  string
	err error
}

// ErrNoConsensus is returned when authorities cannot agree on an IP.
var ErrNoConsensus = errors.New("authorities could not reach consensus on IP address")

// IP returns the public IP address of the current machine by querying
// multiple external authority services concurrently. Returns as soon as
// 2 authorities agree on the same IP address.
func IP(version IPVersion) (string, error) {
	urls, ok := authorities[version]
	if !ok {
		urls = authorities[IPv4]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return queryWithConsensus(ctx, urls)
}

// queryWithConsensus queries all authorities concurrently and returns
// as soon as consensusRequired authorities agree on the same IP.
func queryWithConsensus(ctx context.Context, urls []string) (string, error) {
	results := make(chan result, len(urls))

	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			auth := newAuthority(url)
			ip, err := auth.requestExternalIP(ctx)
			select {
			case results <- result{ip: ip, err: err}:
			case <-ctx.Done():
			}
		}(url)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Track IP counts to find consensus
	ipCounts := make(map[string]int)
	var lastErr error

	for res := range results {
		if res.err != nil {
			lastErr = res.err
			continue
		}

		ipCounts[res.ip]++
		if ipCounts[res.ip] >= consensusRequired {
			return res.ip, nil
		}
	}

	// No consensus reached
	if lastErr != nil {
		return "", lastErr
	}
	return "", ErrNoConsensus
}
