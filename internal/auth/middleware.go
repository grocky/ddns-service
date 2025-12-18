package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/domain"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

// Authenticate validates the API key from the request and returns the authenticated owner.
// It extracts the Bearer token from the Authorization header, validates it against the
// owner specified in the request, and returns an error if authentication fails.
func Authenticate(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	ownerID string,
	repo repository.Repository,
	logger *slog.Logger,
) (*domain.Owner, *response.RequestError) {
	// Extract token from Authorization header
	authHeader := request.Headers["Authorization"]
	if authHeader == "" {
		// Try lowercase (API Gateway normalizes headers)
		authHeader = request.Headers["authorization"]
	}

	token := ExtractBearerToken(authHeader)
	if token == "" {
		logger.Warn("missing or invalid authorization header")
		return nil, &response.RequestError{
			Status:      http.StatusUnauthorized,
			Description: "missing or invalid authorization header",
		}
	}

	// Validate token format
	if !ValidateAPIKeyFormat(token) {
		logger.Warn("invalid API key format")
		return nil, &response.RequestError{
			Status:      http.StatusUnauthorized,
			Description: "invalid API key format",
		}
	}

	// Get owner from repository
	owner, err := repo.GetOwner(ctx, ownerID)
	if err != nil {
		if repository.IsOwnerNotFound(err) {
			logger.Warn("owner not found", "ownerId", ownerID)
			return nil, &response.RequestError{
				Status:      http.StatusUnauthorized,
				Description: "invalid credentials",
			}
		}
		logger.Error("failed to get owner", "error", err)
		return nil, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "authentication failed",
		}
	}

	// Compare API key hash
	providedHash := HashAPIKey(token)
	if !CompareHashes(providedHash, owner.APIKeyHash) {
		logger.Warn("API key mismatch", "ownerId", ownerID)
		return nil, &response.RequestError{
			Status:      http.StatusUnauthorized,
			Description: "invalid credentials",
		}
	}

	logger.Debug("authentication successful", "ownerId", ownerID)
	return owner, nil
}

// AuthenticateAny validates the API key and returns the owner it belongs to.
// Unlike Authenticate, this doesn't require knowing the owner ID upfront.
// It extracts the owner ID from the token validation process.
func AuthenticateAny(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	repo repository.Repository,
	logger *slog.Logger,
) (*domain.Owner, string, *response.RequestError) {
	// Extract token from Authorization header
	authHeader := request.Headers["Authorization"]
	if authHeader == "" {
		authHeader = request.Headers["authorization"]
	}

	token := ExtractBearerToken(authHeader)
	if token == "" {
		logger.Warn("missing or invalid authorization header")
		return nil, "", &response.RequestError{
			Status:      http.StatusUnauthorized,
			Description: "missing or invalid authorization header",
		}
	}

	if !ValidateAPIKeyFormat(token) {
		logger.Warn("invalid API key format")
		return nil, "", &response.RequestError{
			Status:      http.StatusUnauthorized,
			Description: "invalid API key format",
		}
	}

	return nil, token, nil
}
