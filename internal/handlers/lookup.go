package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

// Lookup handles IP lookup requests.
func Lookup(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	repo repository.Repository,
	logger *slog.Logger,
) (response.MappingResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "Lookup")
	defer logger.Info("handler completed", "handler", "Lookup")

	// Extract path parameters
	ownerID := request.PathParameters["ownerId"]
	location := request.PathParameters["location"]

	if ownerID == "" || location == "" {
		logger.Warn("missing path parameters", "ownerId", ownerID, "location", location)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "ownerId and location are required",
		}
	}

	// Lookup in repository
	mapping, err := repo.Get(ctx, ownerID, location)
	if err != nil {
		if repository.IsMappingNotFound(err) {
			logger.Info("mapping not found", "ownerId", ownerID, "location", location)
			return response.MappingResponse{}, &response.RequestError{
				Status:      http.StatusNotFound,
				Description: "mapping not found",
			}
		}

		logger.Error("failed to lookup mapping", "error", err)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to lookup mapping",
		}
	}

	logger.Info("mapping found",
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
