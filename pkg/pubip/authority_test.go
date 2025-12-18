package pubip

import (
	"bytes"
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

// NewTestClient creates a new HTTP client with a custom transport for testing.
func NewTestClient(fn RoundTripFunc) *http.Client {
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
	for _, tt := range requestExternalIPTests {
		client := NewTestClient(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, req.URL.String(), "http://example.com")
			assert.Equal(t, req.Header.Get("User-Agent"), "grocky: pubip")

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(tt.response)),
				Header:     make(http.Header),
			}, nil
		})

		a := authority{client, "http://example.com"}
		actual, err := a.requestExternalIP()
		assert.Assert(t, err == nil)
		assert.Equal(t, tt.expected, actual)
	}
}

func TestErrorRequestExternalIP(t *testing.T) {
	client := NewTestClient(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, req.URL.String(), "http://example.com")
		assert.Equal(t, req.Header.Get("User-Agent"), "grocky: pubip")

		return nil, errors.New("some client error happened")
	})

	a := authority{client, "http://example.com"}
	body, err := a.requestExternalIP()
	assert.Assert(t, body == "")
	// Check that error message contains the expected parts (URL format may vary by Go version)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "http://example.com"))
	assert.Assert(t, strings.Contains(err.Error(), "some client error happened"))
}
