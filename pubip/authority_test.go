package pubip

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"gotest.tools/assert"
)

type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
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
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(tt.response)),
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
	assert.Equal(t, "Get http://example.com: some client error happened", err.Error())
}
