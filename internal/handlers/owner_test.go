package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/auth"
	"github.com/grocky/ddns-service/internal/domain"
	"gotest.tools/assert"
)

// mockEmailService is a mock implementation of email.Service for testing.
type mockEmailService struct {
	sendAPIKeyFunc func(ctx context.Context, toEmail, ownerID, apiKey string) error
}

func (m *mockEmailService) SendAPIKey(ctx context.Context, toEmail, ownerID, apiKey string) error {
	if m.sendAPIKeyFunc != nil {
		return m.sendAPIKeyFunc(ctx, toEmail, ownerID, apiKey)
	}
	return nil
}

// =============================================================================
// CreateOwner Tests
// =============================================================================

func TestCreateOwner_Success(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	var createdOwner domain.Owner
	repo := &mockRepository{
		createOwnerFunc: func(ctx context.Context, owner domain.Owner) error {
			createdOwner = owner
			return nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Body: `{"ownerId":"my-home-lab","email":"user@example.com"}`,
	}

	resp, err := CreateOwner(ctx, request, repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusCreated, resp.Status)
	assert.Equal(t, "my-home-lab", resp.Body.OwnerID)
	assert.Equal(t, "user@example.com", resp.Body.Email)
	assert.Assert(t, strings.HasPrefix(resp.Body.APIKey, auth.APIKeyPrefix), "API key should have correct prefix")
	assert.Assert(t, resp.Body.CreatedAt != "", "CreatedAt should be set")

	// Verify the owner was created with correct data
	assert.Equal(t, "my-home-lab", createdOwner.OwnerID)
	assert.Equal(t, "user@example.com", createdOwner.Email)
	assert.Assert(t, createdOwner.APIKeyHash != "", "API key hash should be set")
}

func TestCreateOwner_EmailNormalized(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	var createdOwner domain.Owner
	repo := &mockRepository{
		createOwnerFunc: func(ctx context.Context, owner domain.Owner) error {
			createdOwner = owner
			return nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Body: `{"ownerId":"test","email":"USER@EXAMPLE.COM"}`,
	}

	resp, err := CreateOwner(ctx, request, repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, "user@example.com", resp.Body.Email)
	assert.Equal(t, "user@example.com", createdOwner.Email)
}

func TestCreateOwner_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Body: `{invalid json}`,
	}

	resp, err := CreateOwner(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "invalid request body", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestCreateOwner_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	testCases := []struct {
		name        string
		body        string
		expectedErr string
	}{
		{
			name:        "missing ownerId",
			body:        `{"email":"user@example.com"}`,
			expectedErr: "ownerId is required",
		},
		{
			name:        "missing email",
			body:        `{"ownerId":"test"}`,
			expectedErr: "email is required",
		},
		{
			name:        "invalid email",
			body:        `{"ownerId":"test","email":"notanemail"}`,
			expectedErr: "invalid email address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := events.APIGatewayProxyRequest{
				Body: tc.body,
			}

			resp, err := CreateOwner(ctx, request, repo, logger)

			assert.Assert(t, err != nil)
			assert.Equal(t, http.StatusBadRequest, err.Status)
			assert.Equal(t, tc.expectedErr, err.Description)
			assert.Equal(t, 0, resp.Status)
		})
	}
}

func TestCreateOwner_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{
		createOwnerFunc: func(ctx context.Context, owner domain.Owner) error {
			return domain.ErrOwnerExists
		},
	}

	request := events.APIGatewayProxyRequest{
		Body: `{"ownerId":"existing-owner","email":"user@example.com"}`,
	}

	resp, err := CreateOwner(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusConflict, err.Status)
	assert.Equal(t, "owner already exists", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestCreateOwner_RepositoryError(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{
		createOwnerFunc: func(ctx context.Context, owner domain.Owner) error {
			return errors.New("database error")
		},
	}

	request := events.APIGatewayProxyRequest{
		Body: `{"ownerId":"test","email":"user@example.com"}`,
	}

	resp, err := CreateOwner(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, "failed to create owner", err.Description)
	assert.Equal(t, 0, resp.Status)
}

// =============================================================================
// RecoverKey Tests
// =============================================================================

func TestRecoverKey_Success(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	var updatedKeyHash string
	var sentEmail, sentOwnerID, sentAPIKey string

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    "test-owner",
				Email:      "user@example.com",
				APIKeyHash: "oldhash",
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
		updateOwnerKeyFunc: func(ctx context.Context, ownerID, newKeyHash string) error {
			updatedKeyHash = newKeyHash
			return nil
		},
	}

	emailSvc := &mockEmailService{
		sendAPIKeyFunc: func(ctx context.Context, toEmail, ownerID, apiKey string) error {
			sentEmail = toEmail
			sentOwnerID = ownerID
			sentAPIKey = apiKey
			return nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Body: `{"email":"user@example.com"}`,
	}

	resp, err := RecoverKey(ctx, request, "test-owner", repo, emailSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "If this email matches our records, a new API key has been sent.", resp.Body.Message)

	// Verify key was updated and email was sent
	assert.Assert(t, updatedKeyHash != "", "Key should be updated")
	assert.Assert(t, updatedKeyHash != "oldhash", "Key hash should change")
	assert.Equal(t, "user@example.com", sentEmail)
	assert.Equal(t, "test-owner", sentOwnerID)
	assert.Assert(t, strings.HasPrefix(sentAPIKey, auth.APIKeyPrefix), "Sent API key should have correct prefix")
}

func TestRecoverKey_EmailCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	var keyUpdated bool

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    "test-owner",
				Email:      "user@example.com", // lowercase
				APIKeyHash: "oldhash",
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
		updateOwnerKeyFunc: func(ctx context.Context, ownerID, newKeyHash string) error {
			keyUpdated = true
			return nil
		},
	}

	emailSvc := &mockEmailService{}

	request := events.APIGatewayProxyRequest{
		Body: `{"email":"USER@EXAMPLE.COM"}`, // uppercase
	}

	resp, err := RecoverKey(ctx, request, "test-owner", repo, emailSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Assert(t, keyUpdated, "Key should be updated even with different case email")
}

func TestRecoverKey_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}
	emailSvc := &mockEmailService{}

	request := events.APIGatewayProxyRequest{
		Body: `{invalid}`,
	}

	resp, err := RecoverKey(ctx, request, "test-owner", repo, emailSvc, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "invalid request body", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRecoverKey_MissingEmail(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}
	emailSvc := &mockEmailService{}

	request := events.APIGatewayProxyRequest{
		Body: `{}`,
	}

	resp, err := RecoverKey(ctx, request, "test-owner", repo, emailSvc, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "email is required", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRecoverKey_OwnerNotFound_NoEnumeration(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return nil, domain.ErrOwnerNotFound
		},
	}

	emailSvc := &mockEmailService{}

	request := events.APIGatewayProxyRequest{
		Body: `{"email":"user@example.com"}`,
	}

	// Should return success to prevent enumeration
	resp, err := RecoverKey(ctx, request, "nonexistent", repo, emailSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "If this email matches our records, a new API key has been sent.", resp.Body.Message)
}

func TestRecoverKey_EmailMismatch_NoEnumeration(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	var keyUpdated bool

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    "test-owner",
				Email:      "real@example.com",
				APIKeyHash: "oldhash",
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
		updateOwnerKeyFunc: func(ctx context.Context, ownerID, newKeyHash string) error {
			keyUpdated = true
			return nil
		},
	}

	emailSvc := &mockEmailService{}

	request := events.APIGatewayProxyRequest{
		Body: `{"email":"wrong@example.com"}`,
	}

	// Should return success to prevent enumeration, but NOT update key
	resp, err := RecoverKey(ctx, request, "test-owner", repo, emailSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Assert(t, !keyUpdated, "Key should NOT be updated for wrong email")
}

// =============================================================================
// RotateKey Tests
// =============================================================================

func TestRotateKey_Success(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	var newKeyHash string

	repo := &mockRepository{
		getOwnerFunc: func(ctx context.Context, ownerID string) (*domain.Owner, error) {
			return &domain.Owner{
				OwnerID:    "test-owner",
				Email:      "user@example.com",
				APIKeyHash: apiKeyHash,
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
		updateOwnerKeyFunc: func(ctx context.Context, ownerID, keyHash string) error {
			newKeyHash = keyHash
			return nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
	}

	resp, err := RotateKey(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "test-owner", resp.Body.OwnerID)
	assert.Assert(t, strings.HasPrefix(resp.Body.APIKey, auth.APIKeyPrefix), "New API key should have correct prefix")
	assert.Assert(t, resp.Body.APIKey != apiKey, "Should return new API key, not old one")
	assert.Assert(t, resp.Body.RotatedAt != "", "RotatedAt should be set")

	// Verify the new hash is different from old
	assert.Assert(t, newKeyHash != apiKeyHash, "Key hash should change")
	assert.Equal(t, auth.HashAPIKey(resp.Body.APIKey), newKeyHash, "Returned key should match stored hash")
}

func TestRotateKey_Unauthorized_MissingAuth(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{},
	}

	resp, err := RotateKey(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, 0, resp.Status)
}

func TestRotateKey_Unauthorized_WrongKey(t *testing.T) {
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
	}

	resp, err := RotateKey(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "invalid credentials", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRotateKey_UpdateError(t *testing.T) {
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
		updateOwnerKeyFunc: func(ctx context.Context, ownerID, keyHash string) error {
			return errors.New("database error")
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
		},
	}

	resp, err := RotateKey(ctx, request, "test-owner", repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, "failed to rotate API key", err.Description)
	assert.Equal(t, 0, resp.Status)
}
