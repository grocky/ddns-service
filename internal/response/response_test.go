package response

import (
	"log/slog"
	"os"
	"testing"

	"gotest.tools/assert"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestRequestError_Error(t *testing.T) {
	err := &RequestError{
		Status:      400,
		Description: "bad request",
	}

	assert.Equal(t, "bad request", err.Error())
}

func TestBuildErrorJSON(t *testing.T) {
	logger := newTestLogger()

	testCases := []struct {
		name        string
		description string
		expected    string
	}{
		{
			name:        "simple message",
			description: "bad request",
			expected:    `{"description":"bad request"}`,
		},
		{
			name:        "message with special characters",
			description: `error: "something" went wrong`,
			expected:    `{"description":"error: \"something\" went wrong"}`,
		},
		{
			name:        "empty message",
			description: "",
			expected:    `{"description":""}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildErrorJSON(tc.description, logger)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestClientIPResponse(t *testing.T) {
	resp := ClientIPResponse{
		Status: 200,
		Body: ClientIPBody{
			PublicIP: "192.168.1.1",
		},
	}

	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "192.168.1.1", resp.Body.PublicIP)
}

func TestMappingResponse(t *testing.T) {
	resp := MappingResponse{
		Status: 200,
		Body: MappingBody{
			OwnerID:   "test-owner",
			Location:  "home",
			IP:        "192.168.1.1",
			UpdatedAt: "2025-01-15T10:30:00Z",
		},
	}

	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "test-owner", resp.Body.OwnerID)
	assert.Equal(t, "home", resp.Body.Location)
	assert.Equal(t, "192.168.1.1", resp.Body.IP)
	assert.Equal(t, "2025-01-15T10:30:00Z", resp.Body.UpdatedAt)
}

func TestOwnerResponse(t *testing.T) {
	resp := OwnerResponse{
		Status: 201,
		Body: OwnerBody{
			OwnerID:   "test-owner",
			Email:     "user@example.com",
			APIKey:    "ddns_sk_abc123",
			CreatedAt: "2025-01-15T10:30:00Z",
		},
	}

	assert.Equal(t, 201, resp.Status)
	assert.Equal(t, "test-owner", resp.Body.OwnerID)
	assert.Equal(t, "user@example.com", resp.Body.Email)
	assert.Equal(t, "ddns_sk_abc123", resp.Body.APIKey)
	assert.Equal(t, "2025-01-15T10:30:00Z", resp.Body.CreatedAt)
}

func TestOwnerResponse_Rotation(t *testing.T) {
	resp := OwnerResponse{
		Status: 200,
		Body: OwnerBody{
			OwnerID:   "test-owner",
			APIKey:    "ddns_sk_newkey",
			RotatedAt: "2025-01-15T10:30:00Z",
		},
	}

	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "test-owner", resp.Body.OwnerID)
	assert.Equal(t, "ddns_sk_newkey", resp.Body.APIKey)
	assert.Equal(t, "2025-01-15T10:30:00Z", resp.Body.RotatedAt)
	assert.Equal(t, "", resp.Body.Email)     // Not set for rotation
	assert.Equal(t, "", resp.Body.CreatedAt) // Not set for rotation
}

func TestMessageResponse(t *testing.T) {
	resp := MessageResponse{
		Status: 200,
		Body: MessageBody{
			Message: "If this email matches our records, a new API key has been sent.",
		},
	}

	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "If this email matches our records, a new API key has been sent.", resp.Body.Message)
}
