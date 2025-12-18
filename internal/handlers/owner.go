package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/auth"
	"github.com/grocky/ddns-service/internal/domain"
	"github.com/grocky/ddns-service/internal/email"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

// CreateOwner handles owner creation requests.
func CreateOwner(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	repo repository.Repository,
	logger *slog.Logger,
) (response.OwnerResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "CreateOwner")
	defer logger.Info("handler completed", "handler", "CreateOwner")

	// Parse request body
	var req domain.CreateOwnerRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		logger.Warn("invalid request body", "error", err)
		return response.OwnerResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "invalid request body",
		}
	}

	// Validate request
	if err := req.Validate(); err != nil {
		logger.Warn("validation failed", "error", err)
		return response.OwnerResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: err.Error(),
		}
	}

	// Generate API key
	apiKey, err := auth.GenerateAPIKey()
	if err != nil {
		logger.Error("failed to generate API key", "error", err)
		return response.OwnerResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to generate API key",
		}
	}

	// Create owner
	now := time.Now().UTC()
	owner := domain.Owner{
		OwnerID:    req.OwnerID,
		Email:      strings.ToLower(req.Email),
		APIKeyHash: auth.HashAPIKey(apiKey),
		CreatedAt:  now,
	}

	if err := repo.CreateOwner(ctx, owner); err != nil {
		if repository.IsOwnerExists(err) {
			logger.Warn("owner already exists", "ownerId", req.OwnerID)
			return response.OwnerResponse{}, &response.RequestError{
				Status:      http.StatusConflict,
				Description: "owner already exists",
			}
		}
		logger.Error("failed to create owner", "error", err)
		return response.OwnerResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to create owner",
		}
	}

	logger.Info("owner created", "ownerId", owner.OwnerID)

	return response.OwnerResponse{
		Status: http.StatusCreated,
		Body: response.OwnerBody{
			OwnerID:   owner.OwnerID,
			Email:     owner.Email,
			APIKey:    apiKey,
			CreatedAt: now.Format(time.RFC3339),
		},
	}, nil
}

// RecoverKey handles API key recovery requests.
func RecoverKey(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	ownerID string,
	repo repository.Repository,
	emailSvc email.Service,
	logger *slog.Logger,
) (response.MessageResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "RecoverKey", "ownerId", ownerID)
	defer logger.Info("handler completed", "handler", "RecoverKey")

	// Parse request body
	var req domain.RecoverKeyRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		logger.Warn("invalid request body", "error", err)
		return response.MessageResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "invalid request body",
		}
	}

	// Validate request
	if err := req.Validate(); err != nil {
		logger.Warn("validation failed", "error", err)
		return response.MessageResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: err.Error(),
		}
	}

	// Always return success to prevent email enumeration
	successMsg := response.MessageResponse{
		Status: http.StatusOK,
		Body: response.MessageBody{
			Message: "If this email matches our records, a new API key has been sent.",
		},
	}

	// Get owner
	owner, err := repo.GetOwner(ctx, ownerID)
	if err != nil {
		if repository.IsOwnerNotFound(err) {
			logger.Info("owner not found for recovery", "ownerId", ownerID)
			return successMsg, nil
		}
		logger.Error("failed to get owner", "error", err)
		return successMsg, nil
	}

	// Check if email matches (case-insensitive)
	if strings.ToLower(req.Email) != strings.ToLower(owner.Email) {
		logger.Info("email mismatch for recovery", "ownerId", ownerID)
		return successMsg, nil
	}

	// Generate new API key
	newAPIKey, err := auth.GenerateAPIKey()
	if err != nil {
		logger.Error("failed to generate new API key", "error", err)
		return successMsg, nil
	}

	// Update the key in the database
	if err := repo.UpdateOwnerKey(ctx, ownerID, auth.HashAPIKey(newAPIKey)); err != nil {
		logger.Error("failed to update owner key", "error", err)
		return successMsg, nil
	}

	// Send email with new key
	if err := emailSvc.SendAPIKey(ctx, owner.Email, ownerID, newAPIKey); err != nil {
		logger.Error("failed to send recovery email", "error", err)
		// Key is already updated, so we still return success
		// The user will need to try recovery again
	}

	logger.Info("API key recovered and sent", "ownerId", ownerID)
	return successMsg, nil
}

// RotateKey handles API key rotation requests.
func RotateKey(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	ownerID string,
	repo repository.Repository,
	logger *slog.Logger,
) (response.OwnerResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "RotateKey", "ownerId", ownerID)
	defer logger.Info("handler completed", "handler", "RotateKey")

	// Authenticate
	_, authErr := auth.Authenticate(ctx, request, ownerID, repo, logger)
	if authErr != nil {
		return response.OwnerResponse{}, authErr
	}

	// Generate new API key
	newAPIKey, err := auth.GenerateAPIKey()
	if err != nil {
		logger.Error("failed to generate new API key", "error", err)
		return response.OwnerResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to generate new API key",
		}
	}

	// Update the key in the database
	if err := repo.UpdateOwnerKey(ctx, ownerID, auth.HashAPIKey(newAPIKey)); err != nil {
		logger.Error("failed to update owner key", "error", err)
		return response.OwnerResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to rotate API key",
		}
	}

	now := time.Now().UTC()
	logger.Info("API key rotated", "ownerId", ownerID)

	return response.OwnerResponse{
		Status: http.StatusOK,
		Body: response.OwnerBody{
			OwnerID:   ownerID,
			APIKey:    newAPIKey,
			RotatedAt: now.Format(time.RFC3339),
		},
	}, nil
}
