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
	getOwnerFunc              func(ctx context.Context, ownerID string) (*domain.Owner, error)
	createOwnerFunc           func(ctx context.Context, owner domain.Owner) error
	updateOwnerKeyFunc        func(ctx context.Context, ownerID, newKeyHash string) error
	putFunc                   func(ctx context.Context, mapping domain.IPMapping) error
	getFunc                   func(ctx context.Context, ownerID, location string) (*domain.IPMapping, error)
	putChallengeFunc          func(ctx context.Context, challenge domain.ACMEChallenge) error
	getChallengeFunc          func(ctx context.Context, ownerID, location string) (*domain.ACMEChallenge, error)
	deleteChallengeFunc       func(ctx context.Context, ownerID, location string) error
	scanExpiredChallengesFunc func(ctx context.Context) ([]domain.ACMEChallenge, error)
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

func (m *mockRepository) PutChallenge(ctx context.Context, challenge domain.ACMEChallenge) error {
	if m.putChallengeFunc != nil {
		return m.putChallengeFunc(ctx, challenge)
	}
	return nil
}

func (m *mockRepository) GetChallenge(ctx context.Context, ownerID, location string) (*domain.ACMEChallenge, error) {
	if m.getChallengeFunc != nil {
		return m.getChallengeFunc(ctx, ownerID, location)
	}
	return nil, domain.ErrChallengeNotFound
}

func (m *mockRepository) DeleteChallenge(ctx context.Context, ownerID, location string) error {
	if m.deleteChallengeFunc != nil {
		return m.deleteChallengeFunc(ctx, ownerID, location)
	}
	return nil
}

func (m *mockRepository) ScanExpiredChallenges(ctx context.Context) ([]domain.ACMEChallenge, error) {
	if m.scanExpiredChallengesFunc != nil {
		return m.scanExpiredChallengesFunc(ctx)
	}
	return nil, nil
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
			assert.Assert(t, mapping.Subdomain != "", "subdomain should be set")
			return nil
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "203.0.113.50",
		},
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "test-owner", resp.Body.OwnerID)
	assert.Equal(t, "home", resp.Body.Location)
	assert.Equal(t, "203.0.113.50", resp.Body.IP)
	assert.Assert(t, resp.Body.Subdomain != "", "subdomain should be in response")
}

func TestRegister_Success_WithSourceIP(t *testing.T) {
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
		RequestContext: events.APIGatewayProxyRequestContext{
			Identity: events.APIGatewayRequestIdentity{
				SourceIP: "192.168.1.100",
			},
		},
		Body: `{"ownerId":"test-owner","location":"office"}`,
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
		Body: `{"location":"home"}`,
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
		Body: `{"ownerId":"test-owner"}`,
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
		Body:    `{"ownerId":"test-owner","location":"home"}`,
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
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, "invalid credentials", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestRegister_NoIP_Available(t *testing.T) {
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
			// No X-Forwarded-For header and no SourceIP
		},
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, domain.ErrMissingIP.Error(), err.Description)
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
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Register(ctx, request, repo, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, "failed to save mapping", err.Description)
	assert.Equal(t, 0, resp.Status)
}
