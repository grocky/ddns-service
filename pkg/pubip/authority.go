package pubip

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// authority is used to query external public IP services.
type authority struct {
	httpClient *http.Client
	url        string
}

// newAuthority creates a new authority with the given URL.
func newAuthority(url string) *authority {
	return &authority{
		url: url,
		httpClient: &http.Client{
			Timeout: time.Second * 3,
		},
	}
}

func (a *authority) requestExternalIP(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create request: %w", err)
	}
	req.Header.Set("User-Agent", "pubip-client/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, a.url)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read body: %w", err)
	}

	return strings.TrimSpace(string(b)), nil
}
