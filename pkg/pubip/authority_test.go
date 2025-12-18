package pubip

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"gotest.tools/assert"
)

// RoundTripFunc is a function type that implements http.RoundTripper.
type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// newTestClient creates a new HTTP client with a custom transport for testing.
func newTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

var requestExternalIPTests = []struct {
	response string
	expected string
}{
	{fmt.Sprintf("%s\n\n", "127.0.0.1"), "127.0.0.1"},
	{fmt.Sprintf(" %s ", "127.0.0.1"), "127.0.0.1"},
	{fmt.Sprintf("\t%s\t", "127.0.0.1"), "127.0.0.1"},
	{fmt.Sprintf("\t%s\n", "127.0.0.1"), "127.0.0.1"},
	{fmt.Sprintf("%s", "127.0.0.1"), "127.0.0.1"},
}

func TestRequestExternalIP(t *testing.T) {
	ctx := context.Background()

	for _, tt := range requestExternalIPTests {
		client := newTestClient(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, req.URL.String(), "http://example.com")
			assert.Equal(t, req.Header.Get("User-Agent"), "pubip-client/1.0")

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(tt.response)),
				Header:     make(http.Header),
			}, nil
		})

		a := authority{client, "http://example.com"}
		actual, err := a.requestExternalIP(ctx)
		assert.Assert(t, err == nil)
		assert.Equal(t, tt.expected, actual)
	}
}

func TestRequestExternalIP_Error(t *testing.T) {
	ctx := context.Background()

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, req.URL.String(), "http://example.com")
		assert.Equal(t, req.Header.Get("User-Agent"), "pubip-client/1.0")

		return nil, errors.New("some client error happened")
	})

	a := authority{client, "http://example.com"}
	body, err := a.requestExternalIP(ctx)
	assert.Assert(t, body == "")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "http://example.com"))
	assert.Assert(t, strings.Contains(err.Error(), "some client error happened"))
}

func TestRequestExternalIP_BadStatusCode(t *testing.T) {
	ctx := context.Background()

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(bytes.NewBufferString("service unavailable")),
			Header:     make(http.Header),
		}, nil
	})

	a := authority{client, "http://example.com"}
	body, err := a.requestExternalIP(ctx)
	assert.Assert(t, body == "")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "503"))
}

func TestRequestExternalIP_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, req.Context().Err()
	})

	a := authority{client, "http://example.com"}
	_, err := a.requestExternalIP(ctx)
	assert.Assert(t, err != nil)
}
