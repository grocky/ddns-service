package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/assert"
)

func TestNew(t *testing.T) {
	cfg := Config{
		APIURL: "https://test.example.com",
		APIKey: "test-key",
	}

	c := New(cfg)

	assert.Equal(t, "https://test.example.com", c.baseURL)
	assert.Equal(t, "test-key", c.apiKey)
}

func TestNew_DefaultURL(t *testing.T) {
	cfg := Config{
		APIKey: "test-key",
	}

	c := New(cfg)

	assert.Equal(t, DefaultAPIURL, c.baseURL)
}

func TestUpdateDNS_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/update", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "ddns-client/1.0", r.Header.Get("User-Agent"))

		// Parse request body
		var req UpdateRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NilError(t, err)
		assert.Equal(t, "test-owner", req.OwnerID)
		assert.Equal(t, "home", req.Location)
		assert.Equal(t, "203.0.113.42", req.IP)

		// Send response
		resp := UpdateResponse{
			OwnerID:   "test-owner",
			Location:  "home",
			IP:        "203.0.113.42",
			Subdomain: "abc12345.grocky.net",
			Changed:   true,
			UpdatedAt: "2025-01-15T10:30:00Z",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(Config{
		APIURL: server.URL,
		APIKey: "test-key",
	})

	resp, err := c.UpdateDNS(context.Background(), "test-owner", "home", "203.0.113.42")

	assert.NilError(t, err)
	assert.Equal(t, "test-owner", resp.OwnerID)
	assert.Equal(t, "home", resp.Location)
	assert.Equal(t, "203.0.113.42", resp.IP)
	assert.Equal(t, "abc12345.grocky.net", resp.Subdomain)
	assert.Equal(t, true, resp.Changed)
}

func TestUpdateDNS_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1800")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ErrorResponse{
			Description: "rate limit exceeded",
		})
	}))
	defer server.Close()

	c := New(Config{
		APIURL: server.URL,
		APIKey: "test-key",
	})

	resp, err := c.UpdateDNS(context.Background(), "test-owner", "home", "203.0.113.42")

	assert.Assert(t, resp == nil)
	assert.Assert(t, err != nil)

	rateLimitErr, ok := err.(*RateLimitError)
	assert.Assert(t, ok, "expected RateLimitError")
	assert.Equal(t, "1800", rateLimitErr.RetryAfter)
}

func TestUpdateDNS_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Description: "invalid API key",
		})
	}))
	defer server.Close()

	c := New(Config{
		APIURL: server.URL,
		APIKey: "invalid-key",
	})

	resp, err := c.UpdateDNS(context.Background(), "test-owner", "home", "203.0.113.42")

	assert.Assert(t, resp == nil)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "invalid API key")
}

func TestUpdateDNS_NoIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify IP is omitted when empty
		var req UpdateRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "", req.IP)

		resp := UpdateResponse{
			OwnerID:   "test-owner",
			Location:  "home",
			IP:        "203.0.113.42", // Server detected IP
			Subdomain: "abc12345.grocky.net",
			Changed:   false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := New(Config{
		APIURL: server.URL,
		APIKey: "test-key",
	})

	// Call without IP (backward compatible mode)
	resp, err := c.UpdateDNS(context.Background(), "test-owner", "home", "")

	assert.NilError(t, err)
	assert.Equal(t, "203.0.113.42", resp.IP)
}

func TestRateLimitError_Error(t *testing.T) {
	err := &RateLimitError{RetryAfter: "300"}
	assert.Equal(t, "rate limited, retry after 300 seconds", err.Error())

	errNoRetry := &RateLimitError{}
	assert.Equal(t, "rate limited", errNoRetry.Error())
}
