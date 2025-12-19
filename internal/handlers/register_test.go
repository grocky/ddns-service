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

// mockRepository is a mock implementation of repository.Repository for testing.
type mockRepository struct {
	getOwnerFunc       func(ctx context.Context, ownerID string) (*domain.Owner, error)
	createOwnerFunc    func(ctx context.Context, owner domain.Owner) error
	updateOwnerKeyFunc func(ctx context.Context, ownerID, newKeyHash string) error
	putFunc            func(ctx context.Context, mapping domain.IPMapping) error
	getFunc            func(ctx context.Context, ownerID, location string) (*domain.IPMapping, error)
}

func (m *mockRepository) GetOwner(ctx context.Context, ownerID string) (*domain.Owner, error) {
	if m.getOwnerFunc != nil {
		return m.getOwnerFunc(ctx, ownerID)
	}
	return nil, domain.ErrOwnerNotFound
}

func (m *mockRepository) CreateOwner(ctx context.Context, owner domain.Owner) error {
	if m.createOwnerFunc != nil {
		return m.createOwnerFunc(ctx, owner)
	}
	return nil
}

func (m *mockRepository) UpdateOwnerKey(ctx context.Context, ownerID, newKeyHash string) error {
	if m.updateOwnerKeyFunc != nil {
		return m.updateOwnerKeyFunc(ctx, ownerID, newKeyHash)
	}
	return nil
}

func (m *mockRepository) Put(ctx context.Context, mapping domain.IPMapping) error {
	if m.putFunc != nil {
		return m.putFunc(ctx, mapping)
	}
	return nil
}

func (m *mockRepository) Get(ctx context.Context, ownerID, location string) (*domain.IPMapping, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, ownerID, location)
	}
	return nil, domain.ErrMappingNotFound
}

func TestRegister_Success_AutoIP(t *testing.T) {
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
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			assert.Equal(t, "test-owner", mapping.OwnerID)
			assert.Equal(t, "home", mapping.LocationName)
			assert.Equal(t, "203.0.113.50", mapping.IP)
			return nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "203.0.113.50",
		},
		Body: `{"ownerId":"test-owner","location":"home","ip":"auto"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "test-owner", resp.Body.OwnerID)
	assert.Equal(t, "home", resp.Body.Location)
	assert.Equal(t, "203.0.113.50", resp.Body.IP)
}

func TestRegister_Success_ExplicitIP(t *testing.T) {
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
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			assert.Equal(t, "192.168.1.100", mapping.IP)
			return nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
		Body: `{"ownerId":"test-owner","location":"office","ip":"192.168.1.100"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "192.168.1.100", resp.Body.IP)
}

func TestRegister_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
		Body: `{invalid json}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "invalid request body", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRegister_MissingOwnerID(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
		Body: `{"location":"home","ip":"auto"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "ownerId is required", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRegister_MissingLocation(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
		Body: `{"ownerId":"test-owner","ip":"auto"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "location is required", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRegister_Unauthorized_MissingAuth(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{},
		Body:    `{"ownerId":"test-owner","location":"home","ip":"auto"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, 0, resp.Status)
}

func TestRegister_Unauthorized_WrongKey(t *testing.T) {
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
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
		Body: `{"ownerId":"test-owner","location":"home","ip":"auto"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "invalid credentials", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRegister_AutoIP_NoForwardedFor(t *testing.T) {
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
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
			// No X-Forwarded-For header
		},
		Body: `{"ownerId":"test-owner","location":"home","ip":"auto"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "could not determine client IP", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRegister_RepositoryError(t *testing.T) {
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
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			return errors.New("database error")
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "203.0.113.50",
		},
		Body: `{"ownerId":"test-owner","location":"home","ip":"auto"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, "failed to save mapping", err.Description)
	assert.Equal(t, 0, resp.Status)
}
