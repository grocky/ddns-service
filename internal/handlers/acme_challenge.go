package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/auth"
	"github.com/grocky/ddns-service/internal/dns"
	"github.com/grocky/ddns-service/internal/domain"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

const (
	// challengeTTLDuration is how long ACME challenges remain valid.
	challengeTTLDuration = 1 * time.Hour
)

// CreateACMEChallenge handles POST /acme-challenge requests.
// It creates a TXT record for DNS-01 ACME challenges.
func CreateACMEChallenge(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	repo repository.Repository,
	dnsService dns.Service,
	logger *slog.Logger,
) (response.ACMEChallengeResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "CreateACMEChallenge")
	defer logger.Info("handler completed", "handler", "CreateACMEChallenge")

	now := time.Now().UTC()

	// Parse request body
	var req domain.CreateChallengeRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		logger.Warn("invalid request body", "error", err)
		return response.ACMEChallengeResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "invalid request body",
		}
	}

	// Validate request
	if err := req.Validate(); err != nil {
		logger.Warn("validation failed", "error", err)
		return response.ACMEChallengeResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: err.Error(),
		}
	}

	// Authenticate - verify API key matches the ownerId in the request
	_, authErr := auth.Authenticate(ctx, request, req.OwnerID, repo, logger)
	if authErr != nil {
		return response.ACMEChallengeResponse{}, authErr
	}

	// Verify owner has an IP mapping for this location (proves ownership)
	mapping, err := repo.Get(ctx, req.OwnerID, req.Location)
	if err != nil {
		if repository.IsMappingNotFound(err) {
			logger.Warn("no IP mapping for location", "ownerId", req.OwnerID, "location", req.Location)
			return response.ACMEChallengeResponse{}, &response.RequestError{
				Status:      http.StatusForbidden,
				Description: "no IP mapping exists for this location",
			}
		}
		logger.Error("failed to get mapping", "error", err)
		return response.ACMEChallengeResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to verify ownership",
		}
	}

	// Check if challenge already exists
	existing, err := repo.GetChallenge(ctx, req.OwnerID, req.Location)
	if err == nil && existing != nil {
		logger.Warn("challenge already exists", "ownerId", req.OwnerID, "location", req.Location)
		return response.ACMEChallengeResponse{}, &response.RequestError{
			Status:      http.StatusConflict,
			Description: "challenge already exists, delete it first",
		}
	}

	// Build TXT record name
	subdomain := mapping.Subdomain
	txtRecordName := dns.BuildACMEChallengeName(subdomain)
	fullTxtRecord := dns.FormatFQDN(txtRecordName)

	// Create Route53 TXT record
	if err := dnsService.UpsertTXTRecord(ctx, txtRecordName, req.TxtValue); err != nil {
		logger.Error("failed to create TXT record", "error", err)
		return response.ACMEChallengeResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to create DNS record",
		}
	}

	// Store challenge in DynamoDB
	expiresAt := now.Add(challengeTTLDuration)
	challenge := domain.ACMEChallenge{
		OwnerID:      req.OwnerID,
		LocationName: req.Location,
		Subdomain:    subdomain,
		TxtValue:     req.TxtValue,
		TxtRecord:    fullTxtRecord,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
		TTL:          expiresAt.Unix(),
	}

	if err := repo.PutChallenge(ctx, challenge); err != nil {
		logger.Error("failed to save challenge", "error", err)
		// Try to clean up the DNS record
		_ = dnsService.DeleteTXTRecord(ctx, txtRecordName, req.TxtValue)
		return response.ACMEChallengeResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to save challenge",
		}
	}

	logger.Info("ACME challenge created",
		"ownerId", req.OwnerID,
		"location", req.Location,
		"subdomain", subdomain,
		"txtRecord", fullTxtRecord,
	)

	return response.ACMEChallengeResponse{
		Status: http.StatusCreated,
		Body: response.ACMEChallengeBody{
			OwnerID:   challenge.OwnerID,
			Location:  challenge.LocationName,
			Subdomain: dns.FormatFQDN(subdomain),
			TxtRecord: fullTxtRecord,
			TxtValue:  challenge.TxtValue,
			CreatedAt: challenge.CreatedAt.Format(time.RFC3339),
			ExpiresAt: challenge.ExpiresAt.Format(time.RFC3339),
		},
	}, nil
}

// DeleteACMEChallenge handles DELETE /acme-challenge requests.
// It removes the TXT record after certificate issuance.
func DeleteACMEChallenge(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	repo repository.Repository,
	dnsService dns.Service,
	logger *slog.Logger,
) (response.ACMEDeleteResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "DeleteACMEChallenge")
	defer logger.Info("handler completed", "handler", "DeleteACMEChallenge")

	// Parse request body
	var req domain.DeleteChallengeRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		logger.Warn("invalid request body", "error", err)
		return response.ACMEDeleteResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "invalid request body",
		}
	}

	// Validate request
	if err := req.Validate(); err != nil {
		logger.Warn("validation failed", "error", err)
		return response.ACMEDeleteResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: err.Error(),
		}
	}

	// Authenticate - verify API key matches the ownerId in the request
	_, authErr := auth.Authenticate(ctx, request, req.OwnerID, repo, logger)
	if authErr != nil {
		return response.ACMEDeleteResponse{}, authErr
	}

	// Get existing challenge
	challenge, err := repo.GetChallenge(ctx, req.OwnerID, req.Location)
	if err != nil {
		if errors.Is(err, domain.ErrChallengeNotFound) {
			return response.ACMEDeleteResponse{}, &response.RequestError{
				Status:      http.StatusNotFound,
				Description: "no active challenge for this location",
			}
		}
		logger.Error("failed to get challenge", "error", err)
		return response.ACMEDeleteResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to get challenge",
		}
	}

	// Delete Route53 TXT record
	txtRecordName := dns.BuildACMEChallengeName(challenge.Subdomain)
	if err := dnsService.DeleteTXTRecord(ctx, txtRecordName, challenge.TxtValue); err != nil {
		logger.Error("failed to delete TXT record", "error", err)
		// Continue anyway - we'll clean up orphaned records later via scheduled cleanup
	}

	// Delete from DynamoDB
	if err := repo.DeleteChallenge(ctx, req.OwnerID, req.Location); err != nil {
		logger.Error("failed to delete challenge from DB", "error", err)
		return response.ACMEDeleteResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to delete challenge",
		}
	}

	logger.Info("ACME challenge deleted",
		"ownerId", req.OwnerID,
		"location", req.Location,
	)

	return response.ACMEDeleteResponse{
		Status: http.StatusOK,
		Body: response.ACMEDeleteBody{
			OwnerID:   challenge.OwnerID,
			Location:  challenge.LocationName,
			Subdomain: dns.FormatFQDN(challenge.Subdomain),
			TxtRecord: challenge.TxtRecord,
			Deleted:   true,
		},
	}, nil
}
