package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/auth"
	"github.com/grocky/ddns-service/internal/domain"
	"gotest.tools/assert"
)

func TestLookup_Success(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)
	updatedAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    "test-owner",
				Email:      "user@example.com",
				APIKeyHash: apiKeyHash,
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
		getFunc: func(ctx context.Context, ownerID, location string) (*domain.IPMapping, error) {
			assert.Equal(t, "test-owner", ownerID)
			assert.Equal(t, "home", location)
			return &domain.IPMapping{
				OwnerID:      "test-owner",
				LocationName: "home",
				IP:           "203.0.113.50",
				UpdatedAt:    updatedAt,
			}, nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Path: "/lookup/test-owner/home",
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
	}

	resp, err := Lookup(ctx, request, repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "test-owner", resp.Body.OwnerID)
	assert.Equal(t, "home", resp.Body.Location)
	assert.Equal(t, "203.0.113.50", resp.Body.IP)
	assert.Equal(t, "2025-01-15T10:30:00Z", resp.Body.UpdatedAt)
}

func TestLookup_MissingPathParameters(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	testCases := []struct {
		name string
		path string
	}{
		{"empty path", "/"},
		{"only lookup", "/lookup"},
		{"only owner", "/lookup/test-owner"},
		{"missing owner", "/lookup//home"},
		{"missing location", "/lookup/test-owner/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := events.APIGatewayProxyRequest{
				Path: tc.path,
				Headers: map[string]string{
					"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
				},
			}

			resp, err := Lookup(ctx, request, repo, logger)

			assert.Assert(t, err != nil)
			assert.Equal(t, http.StatusBadRequest, err.Status)
			assert.Equal(t, "ownerId and location are required", err.Description)
			assert.Equal(t, 0, resp.Status)
		})
	}
}

func TestLookup_Unauthorized_MissingAuth(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Path:    "/lookup/test-owner/home",
		Headers: map[string]string{},
	}

	resp, err := Lookup(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, 0, resp.Status)
}

func TestLookup_Unauthorized_WrongKey(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    "test-owner",
				Email:      "user@example.com",
				APIKeyHash: auth.HashAPIKey("ddns_sk_correctkey"),
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Path: "/lookup/test-owner/home",
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
	}

	resp, err := Lookup(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "invalid credentials", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestLookup_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    "test-owner",
				Email:      "user@example.com",
				APIKeyHash: apiKeyHash,
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
		getFunc: func(ctx context.Context, ownerID, location string) (*domain.IPMapping, error) {
			return nil, domain.ErrMappingNotFound
		},
	}

	request := events.APIGatewayProxyRequest{
		Path: "/lookup/test-owner/nonexistent",
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
	}

	resp, err := Lookup(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusNotFound, err.Status)
	assert.Equal(t, "mapping not found", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestLookup_RepositoryError(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    "test-owner",
				Email:      "user@example.com",
				APIKeyHash: apiKeyHash,
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
		getFunc: func(ctx context.Context, ownerID, location string) (*domain.IPMapping, error) {
			return nil, errors.New("database error")
		},
	}

	request := events.APIGatewayProxyRequest{
		Path: "/lookup/test-owner/home",
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
	}

	resp, err := Lookup(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, "failed to lookup mapping", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestLookup_PathParsing(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	// Track what owner/location we receive
	var receivedOwner, receivedLocation string

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    ownerID,
				Email:      "user@example.com",
				APIKeyHash: apiKeyHash,
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
		getFunc: func(ctx context.Context, ownerID, location string) (*domain.IPMapping, error) {
			receivedOwner = ownerID
			receivedLocation = location
			return &domain.IPMapping{
				OwnerID:      ownerID,
				LocationName: location,
				IP:           "192.168.1.1",
				UpdatedAt:    time.Now().UTC(),
			}, nil
		},
	}

	testCases := []struct {
		path             string
		expectedOwner    string
		expectedLocation string
	}{
		{"/lookup/my-home-lab/office", "my-home-lab", "office"},
		{"/lookup/owner123/location456", "owner123", "location456"},
		{"/lookup/a/b", "a", "b"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			request := events.APIGatewayProxyRequest{
				Path: tc.path,
				Headers: map[string]string{
					"Authorization": "Bearer " + apiKey,
				},
			}

			_, err := Lookup(ctx, request, repo, logger)
			assert.Assert(t, err == nil)
			assert.Equal(t, tc.expectedOwner, receivedOwner)
			assert.Equal(t, tc.expectedLocation, receivedLocation)
		})
	}
}
