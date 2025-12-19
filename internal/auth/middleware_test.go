package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/domain"
	"gotest.tools/assert"
)

// mockRepository is a mock implementation of repository.Repository for testing.
type mockRepository struct {
	getOwnerFunc      func(ctx context.Context, ownerID string) (*domain.Owner, error)
	createOwnerFunc   func(ctx context.Context, owner domain.Owner) error
	updateOwnerKeyFunc func(ctx context.Context, ownerID, newKeyHash string) error
	putFunc           func(ctx context.Context, mapping domain.IPMapping) error
	getFunc           func(ctx context.Context, ownerID, location string) (*domain.IPMapping, error)
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

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestAuthenticate_Success(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	// Generate a real API key and its hash
	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := HashAPIKey(apiKey)

	owner := &domain.Owner{
		OwnerID:    "test-owner",
		Email:      "user@example.com",
		APIKeyHash: apiKeyHash,
		CreatedAt:  time.Now().UTC(),
	}

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			if ownerID == "test-owner" {
				return owner, nil
			}
			return nil, domain.ErrOwnerNotFound
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
	}

	result, err := Authenticate(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, err == nil, "expected no error, got %v", err)
	assert.Equal(t, owner.OwnerID, result.OwnerID)
	assert.Equal(t, owner.Email, result.Email)
}

func TestAuthenticate_LowercaseHeader(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := HashAPIKey(apiKey)

	owner := &domain.Owner{
		OwnerID:    "test-owner",
		Email:      "user@example.com",
		APIKeyHash: apiKeyHash,
		CreatedAt:  time.Now().UTC(),
	}

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return owner, nil
		},
	}

	// API Gateway sometimes normalizes headers to lowercase
	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"authorization": "Bearer " + apiKey,
		},
	}

	result, err := Authenticate(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, owner.OwnerID, result.OwnerID)
}

func TestAuthenticate_MissingAuthHeader(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{},
	}

	result, err := Authenticate(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, result == nil)
	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "missing or invalid authorization header", err.Description)
}

func TestAuthenticate_EmptyAuthHeader(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "",
		},
	}

	result, err := Authenticate(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, result == nil)
	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
}

func TestAuthenticate_InvalidBearerFormat(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	testCases := []struct {
		name       string
		authHeader string
	}{
		{"no bearer prefix", "ddns_sk_somekey"},
		{"basic auth", "Basic dXNlcjpwYXNz"},
		{"lowercase bearer", "bearer ddns_sk_somekey"},
		{"bearer only", "Bearer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := events.APIGatewayProxyRequest{
				Headers: map[string]string{
					"Authorization": tc.authHeader,
				},
			}

			result, err := Authenticate(ctx, request, "test-owner", repo, logger)

			assert.Assert(t, result == nil)
			assert.Assert(t, err != nil)
			assert.Equal(t, http.StatusUnauthorized, err.Status)
		})
	}
}

func TestAuthenticate_InvalidAPIKeyFormat(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer invalid_prefix_key",
		},
	}

	result, err := Authenticate(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, result == nil)
	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "invalid API key format", err.Description)
}

func TestAuthenticate_OwnerNotFound(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return nil, domain.ErrOwnerNotFound
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
	}

	result, err := Authenticate(ctx, request, "nonexistent", repo, logger)

	assert.Assert(t, result == nil)
	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "invalid credentials", err.Description)
}

func TestAuthenticate_RepositoryError(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return nil, errors.New("database connection failed")
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
	}

	result, err := Authenticate(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, result == nil)
	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, "authentication failed", err.Description)
}

func TestAuthenticate_APIKeyMismatch(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	// Owner has a different API key hash
	owner := &domain.Owner{
		OwnerID:    "test-owner",
		Email:      "user@example.com",
		APIKeyHash: HashAPIKey("ddns_sk_differentkey"),
		CreatedAt:  time.Now().UTC(),
	}

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return owner, nil
		},
	}

	// Provide a different key
	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
	}

	result, err := Authenticate(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, result == nil)
	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "invalid credentials", err.Description)
}

func TestAuthenticateAny_Success(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}
	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
	}

	_, token, err := AuthenticateAny(ctx, request, repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, apiKey, token)
}

func TestAuthenticateAny_MissingHeader(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{},
	}

	_, token, err := AuthenticateAny(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, "", token)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
}

func TestAuthenticateAny_InvalidFormat(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer invalid_key_format",
		},
	}

	_, token, err := AuthenticateAny(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, "", token)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "invalid API key format", err.Description)
}
