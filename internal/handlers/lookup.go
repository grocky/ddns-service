package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/auth"
	"github.com/grocky/ddns-service/internal/dns"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

// Lookup handles IP lookup requests.
// Path format: /lookup/{ownerId}/{location}
// Requires authentication via API key.
func Lookup(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	repo repository.Repository,
	logger *slog.Logger,
) (response.MappingResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "Lookup")
	defer logger.Info("handler completed", "handler", "Lookup")

	// Parse path: /lookup/{ownerId}/{location}
	// Remove leading slash and split
	path := strings.TrimPrefix(request.Path, "/")
	parts := strings.Split(path, "/")

	var ownerID, location string
	if len(parts) >= 3 {
		ownerID = parts[1]
		location = parts[2]
	}

	if ownerID == "" || location == "" {
		logger.Warn("missing path parameters", "ownerId", ownerID, "location", location, "path", request.Path)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "ownerId and location are required",
		}
	}

	// Authenticate - verify API key matches the ownerId in the path
	_, authErr := auth.Authenticate(ctx, request, ownerID, repo, logger)
	if authErr != nil {
		return response.MappingResponse{}, authErr
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

	// Use stored subdomain if set, otherwise generate hash
	subdomain := mapping.Subdomain
	if subdomain == "" {
		subdomain = dns.GenerateSubdomain(mapping.OwnerID, mapping.LocationName)
	}
	fullSubdomain := dns.FormatFQDN(subdomain)

	logger.Info("mapping found",
		"ownerId", mapping.OwnerID,
		"location", mapping.LocationName,
		"ip", mapping.IP,
		"subdomain", fullSubdomain,
	)

	return response.MappingResponse{
		Status: http.StatusOK,
		Body: response.MappingBody{
			OwnerID:   mapping.OwnerID,
			Location:  mapping.LocationName,
			IP:        mapping.IP,
			Subdomain: fullSubdomain,
			UpdatedAt: mapping.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}
