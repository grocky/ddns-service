package pubip

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// authority is used to query external public IP services.
type authority struct {
	httpClient *http.Client
	domain     string
}

// newAuthority creates a new authority with the given domain.
func newAuthority(domain string) *authority {
	return &authority{
		domain: domain,
		httpClient: &http.Client{
			Timeout: time.Second * 2,
		},
	}
}

func (a *authority) requestExternalIP() (string, error) {
	req, err := http.NewRequest(http.MethodGet, a.domain, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create request: %w", err)
	}
	req.Header.Set("User-Agent", "grocky: pubip")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read body: %w", err)
	}

	return strings.TrimSpace(string(b)), nil
}
