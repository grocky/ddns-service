package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultAPIURL  = "https://ddns.grocky.net"
	DefaultTimeout = 30 * time.Second
)

// Client is the DDNS API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// New creates a new DDNS API client.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	baseURL := cfg.APIURL
	if baseURL == "" {
		baseURL = DefaultAPIURL
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
	}
}

// RateLimitError indicates the request was rate limited.
type RateLimitError struct {
	RetryAfter string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("rate limited, retry after %s seconds", e.RetryAfter)
	}
	return "rate limited"
}

// UpdateDNS sends an update request to the DDNS server.
// If ip is non-empty, it will be sent to the server as the client-detected IP.
func (c *Client) UpdateDNS(ctx context.Context, owner, location, ip string) (*UpdateResponse, error) {
	req := UpdateRequest{
		OwnerID:  owner,
		Location: location,
		IP:       ip,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/update", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("User-Agent", "ddns-client/1.0")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Description != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Description)
		}
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var updateResp UpdateResponse
	if err := json.Unmarshal(respBody, &updateResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &updateResp, nil
}
