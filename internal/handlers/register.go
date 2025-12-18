package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/auth"
	"github.com/grocky/ddns-service/internal/domain"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

// Register handles IP registration requests.
// Requires authentication via API key.
func Register(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	repo repository.Repository,
	logger *slog.Logger,
) (response.MappingResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "Register")
	defer logger.Info("handler completed", "handler", "Register")

	// Parse request body first to get ownerId for authentication
	var req domain.RegisterRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		logger.Warn("invalid request body", "error", err)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "invalid request body",
		}
	}

	// Validate request
	if err := req.Validate(); err != nil {
		logger.Warn("validation failed", "error", err)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: err.Error(),
		}
	}

	// Authenticate - verify API key matches the ownerId in the request
	_, authErr := auth.Authenticate(ctx, request, req.OwnerID, repo, logger)
	if authErr != nil {
		return response.MappingResponse{}, authErr
	}

	// Determine IP address
	ip := req.IP
	if ip == "" || ip == "auto" {
		ip = request.Headers["X-Forwarded-For"]
		if ip == "" {
			logger.Warn("could not determine client IP")
			return response.MappingResponse{}, &response.RequestError{
				Status:      http.StatusBadRequest,
				Description: "could not determine client IP",
			}
		}
	}

	// Create mapping
	mapping := domain.IPMapping{
		OwnerID:      req.OwnerID,
		LocationName: req.Location,
		IP:           ip,
		UpdatedAt:    time.Now().UTC(),
	}

	// Save to repository
	if err := repo.Put(ctx, mapping); err != nil {
		logger.Error("failed to save mapping", "error", err)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to save mapping",
		}
	}

	logger.Info("mapping registered",
		"ownerId", mapping.OwnerID,
		"location", mapping.LocationName,
		"ip", mapping.IP,
	)

	return response.MappingResponse{
		Status: http.StatusOK,
		Body: response.MappingBody{
			OwnerID:   mapping.OwnerID,
			Location:  mapping.LocationName,
			IP:        mapping.IP,
			UpdatedAt: mapping.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}
