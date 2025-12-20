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
	"github.com/grocky/ddns-service/internal/ratelimit"
	"gotest.tools/assert"
)

// mockDNSService is a mock implementation of dns.Service for testing.
type mockDNSService struct {
	upsertRecordFunc    func(ctx context.Context, subdomain, ip string) error
	deleteRecordFunc    func(ctx context.Context, subdomain string) error
	upsertTXTRecordFunc func(ctx context.Context, name, value string) error
	deleteTXTRecordFunc func(ctx context.Context, name, value string) error
}

func (m *mockDNSService) UpsertRecord(ctx context.Context, subdomain, ip string) error {
	if m.upsertRecordFunc != nil {
		return m.upsertRecordFunc(ctx, subdomain, ip)
	}
	return nil
}

func (m *mockDNSService) DeleteRecord(ctx context.Context, subdomain string) error {
	if m.deleteRecordFunc != nil {
		return m.deleteRecordFunc(ctx, subdomain)
	}
	return nil
}

func (m *mockDNSService) UpsertTXTRecord(ctx context.Context, name, value string) error {
	if m.upsertTXTRecordFunc != nil {
		return m.upsertTXTRecordFunc(ctx, name, value)
	}
	return nil
}

func (m *mockDNSService) DeleteTXTRecord(ctx context.Context, name, value string) error {
	if m.deleteTXTRecordFunc != nil {
		return m.deleteTXTRecordFunc(ctx, name, value)
	}
	return nil
}

func TestUpdate_NewMapping(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	var savedMapping domain.IPMapping
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
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			savedMapping = mapping
			return nil
		},
	}

	dnsSvc := &mockDNSService{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "203.0.113.50",
		},
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "test-owner", resp.Body.OwnerID)
	assert.Equal(t, "home", resp.Body.Location)
	assert.Equal(t, "203.0.113.50", resp.Body.IP)
	assert.Assert(t, resp.Body.Changed, "should indicate IP changed for new mapping")
	assert.Assert(t, resp.Body.Subdomain != "", "subdomain should be set")
	assert.Equal(t, 1, savedMapping.HourlyChangeCount)
}

func TestUpdate_IPUnchanged(t *testing.T) {
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
			return &domain.IPMapping{
				OwnerID:      "test-owner",
				LocationName: "home",
				IP:           "203.0.113.50", // Same IP as request
				Subdomain:    "a3f8c2d1",
				UpdatedAt:    time.Now().UTC(),
			}, nil
		},
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			t.Fatal("should not save mapping when IP unchanged")
			return nil
		},
	}

	dnsSvc := &mockDNSService{
		upsertRecordFunc: func(ctx context.Context, subdomain, ip string) error {
			t.Fatal("should not update DNS when IP unchanged")
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

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Assert(t, !resp.Body.Changed, "should indicate IP not changed")
	assert.Equal(t, "203.0.113.50", resp.Body.IP)
}

func TestUpdate_IPChanged(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	var dnsUpdated bool
	var savedMapping domain.IPMapping

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
			return &domain.IPMapping{
				OwnerID:           "test-owner",
				LocationName:      "home",
				IP:                "192.168.1.100", // Old IP
				Subdomain:         "a3f8c2d1",
				UpdatedAt:         time.Now().UTC(),
				LastIPChangeAt:    time.Now().Add(-2 * time.Hour),
				HourlyChangeCount: 1,
			}, nil
		},
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			savedMapping = mapping
			return nil
		},
	}

	dnsSvc := &mockDNSService{
		upsertRecordFunc: func(ctx context.Context, subdomain, ip string) error {
			dnsUpdated = true
			assert.Equal(t, "203.0.113.50", ip)
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

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Assert(t, resp.Body.Changed, "should indicate IP changed")
	assert.Equal(t, "203.0.113.50", resp.Body.IP)
	assert.Assert(t, dnsUpdated, "DNS should have been updated")
	assert.Equal(t, "203.0.113.50", savedMapping.IP)
}

func TestUpdate_RateLimitExceeded(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)
	now := time.Now().UTC()

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
			return &domain.IPMapping{
				OwnerID:           "test-owner",
				LocationName:      "home",
				IP:                "192.168.1.100",
				Subdomain:         "a3f8c2d1",
				UpdatedAt:         now,
				LastIPChangeAt:    now.Add(-time.Minute), // Changed recently
				HourlyChangeCount: ratelimit.MaxChangesPerHour,
			}, nil
		},
	}

	dnsSvc := &mockDNSService{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "203.0.113.50", // Different IP
		},
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusTooManyRequests, err.Status)
	assert.Equal(t, domain.ErrRateLimitExceeded.Error(), err.Description)
	assert.Assert(t, err.RetryAfter > 0, "should have retry after")
	assert.Equal(t, 0, resp.Status)
}

func TestUpdate_DNSError(t *testing.T) {
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

	dnsSvc := &mockDNSService{
		upsertRecordFunc: func(ctx context.Context, subdomain, ip string) error {
			return errors.New("route53 error")
		},
	}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "203.0.113.50",
		},
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusInternalServerError, err.Status)
	assert.Equal(t, "failed to update DNS record", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestUpdate_MissingIP(t *testing.T) {
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

	dnsSvc := &mockDNSService{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + apiKey,
			// No X-Forwarded-For or SourceIP
		},
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, domain.ErrMissingIP.Error(), err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestUpdate_Unauthorized(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}
	dnsSvc := &mockDNSService{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{},
		Body:    `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusUnauthorized, err.Status)
	assert.Equal(t, 0, resp.Status)
}

func TestUpdate_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	repo := &mockRepository{}
	dnsSvc := &mockDNSService{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop",
		},
		Body: `{invalid json}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "invalid request body", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestUpdate_MultipleIPsInForwardedFor(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	var savedIP string
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
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			savedIP = mapping.IP
			return nil
		},
	}

	dnsSvc := &mockDNSService{}

	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "203.0.113.50, 10.0.0.1, 192.168.1.1",
		},
		Body: `{"ownerId":"test-owner","location":"home"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	// Should use the first IP in the list
	assert.Equal(t, "203.0.113.50", savedIP)
	assert.Equal(t, "203.0.113.50", resp.Body.IP)
}

func TestUpdate_ClientProvidedIP(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	var savedIP string
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
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			savedIP = mapping.IP
			return nil
		},
	}

	dnsSvc := &mockDNSService{}

	// Client provides IP in request body
	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "10.0.0.1", // Server detects different IP
		},
		Body: `{"ownerId":"test-owner","location":"home","ip":"203.0.113.42"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	// Should use the client-provided IP, not the server-detected one
	assert.Equal(t, "203.0.113.42", savedIP)
	assert.Equal(t, "203.0.113.42", resp.Body.IP)
}

func TestUpdate_ClientProvidedIP_Invalid(t *testing.T) {
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

	dnsSvc := &mockDNSService{}

	// Client provides invalid IP
	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "10.0.0.1",
		},
		Body: `{"ownerId":"test-owner","location":"home","ip":"not-an-ip"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err != nil)
	assert.Equal(t, http.StatusBadRequest, err.Status)
	assert.Equal(t, "invalid IP address format", err.Description)
	assert.Equal(t, 0, resp.Status)
}

func TestUpdate_ClientProvidedIPv6(t *testing.T) {
	ctx := context.Background()
	logger := newTestLogger()

	apiKey := "ddns_sk_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop"
	apiKeyHash := auth.HashAPIKey(apiKey)

	var savedIP string
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
		putFunc: func(ctx context.Context, mapping domain.IPMapping) error {
			savedIP = mapping.IP
			return nil
		},
	}

	dnsSvc := &mockDNSService{}

	// Client provides IPv6 address
	request := events.APIGatewayProxyRequest{
		Headers: map[string]string{
			"Authorization":   "Bearer " + apiKey,
			"X-Forwarded-For": "10.0.0.1",
		},
		Body: `{"ownerId":"test-owner","location":"home","ip":"2001:db8::1"}`,
	}

	resp, err := Update(ctx, request, repo, dnsSvc, logger)

	assert.Assert(t, err == nil)
	assert.Equal(t, http.StatusOK, resp.Status)
	assert.Equal(t, "2001:db8::1", savedIP)
	assert.Equal(t, "2001:db8::1", resp.Body.IP)
}
