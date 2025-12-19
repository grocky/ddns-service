package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"gotest.tools/assert"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestGetPublicIP_SourceIP(t *testing.T) {
	logger := newTestLogger()

	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Identity: events.APIGatewayRequestIdentity{
				SourceIP: "203.0.113.50",
			},
		},
		Headers: map[string]string{},
	}

	resp, err := GetPublicIP(request, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "203.0.113.50", resp.Body.PublicIP)
}

func TestGetPublicIP_XForwardedFor_Fallback(t *testing.T) {
	logger := newTestLogger()

	// SourceIP is empty, should fall back to X-Forwarded-For
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Identity: events.APIGatewayRequestIdentity{
				SourceIP: "",
			},
		},
		Headers: map[string]string{
			"X-Forwarded-For": "198.51.100.25",
		},
	}

	resp, err := GetPublicIP(request, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "198.51.100.25", resp.Body.PublicIP)
}

func TestGetPublicIP_SourceIP_Preferred(t *testing.T) {
	logger := newTestLogger()

	// Both SourceIP and X-Forwarded-For are present, SourceIP should be preferred
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Identity: events.APIGatewayRequestIdentity{
				SourceIP: "203.0.113.50",
			},
		},
		Headers: map[string]string{
			"X-Forwarded-For": "198.51.100.25",
		},
	}

	resp, err := GetPublicIP(request, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "203.0.113.50", resp.Body.PublicIP)
}

func TestGetPublicIP_NoIP(t *testing.T) {
	logger := newTestLogger()

	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Identity: events.APIGatewayRequestIdentity{
				SourceIP: "",
			},
		},
		Headers: map[string]string{},
	}

	resp, err := GetPublicIP(request, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "Client IP not found", err.Description)
	assert.Equal(t, 0, resp.Status) // Zero value response
}

func TestGetPublicIP_IPv6(t *testing.T) {
	logger := newTestLogger()

	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			Identity: events.APIGatewayRequestIdentity{
				SourceIP: "2001:db8::1",
			},
		},
		Headers: map[string]string{},
	}

	resp, err := GetPublicIP(request, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "2001:db8::1", resp.Body.PublicIP)
}
