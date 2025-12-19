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
	"github.com/grocky/ddns-service/internal/dns"
	"github.com/grocky/ddns-service/internal/domain"
	"github.com/grocky/ddns-service/internal/ratelimit"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

// Update handles IP update requests.
// This is the main endpoint for DDNS clients to poll.
// IP is automatically detected from the request context.
// Rate limited to 2 IP changes per hour.
func Update(
	ctx context.Context,
	request events.APIGatewayProxyRequest,
	repo repository.Repository,
	dnsService dns.Service,
	logger *slog.Logger,
) (response.MappingResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "Update")
	defer logger.Info("handler completed", "handler", "Update")

	now := time.Now().UTC()

	// Parse request body
	var req domain.UpdateRequest
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

	// Get client IP from request
	ip := extractClientIP(request)
	if ip == "" {
		logger.Warn("could not determine client IP")
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: domain.ErrMissingIP.Error(),
		}
	}

	logger.Info("client IP detected", "ip", ip)

	// Get existing mapping (may not exist yet)
	existing, err := repo.Get(ctx, req.OwnerID, req.Location)
	if err != nil && !repository.IsMappingNotFound(err) {
		logger.Error("failed to get mapping", "error", err)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to get mapping",
		}
	}

	// Check if this is a new mapping or if IP has changed
	isNew := existing == nil
	ipChanged := isNew || existing.IP != ip

	// Determine subdomain: use existing custom subdomain or generate hash
	var subdomain string
	if existing != nil && existing.Subdomain != "" {
		subdomain = existing.Subdomain
	} else {
		subdomain = dns.GenerateSubdomain(req.OwnerID, req.Location)
	}
	fullSubdomain := dns.FormatFQDN(subdomain)

	if !ipChanged {
		// IP hasn't changed - just return current state
		logger.Info("IP unchanged, no update needed",
			"ownerId", req.OwnerID,
			"location", req.Location,
			"ip", ip,
		)
		return response.MappingResponse{
			Status: http.StatusOK,
			Body: response.MappingBody{
				OwnerID:   existing.OwnerID,
				Location:  existing.LocationName,
				IP:        existing.IP,
				Subdomain: fullSubdomain,
				Changed:   false,
				UpdatedAt: existing.UpdatedAt.Format(time.RFC3339),
			},
		}, nil
	}

	// IP has changed - check rate limit
	var mappingForRateLimit *domain.IPMapping
	if existing != nil {
		mappingForRateLimit = existing
	}

	rateLimitResult := ratelimit.Check(mappingForRateLimit, now)
	if !rateLimitResult.Allowed {
		retryAfterSeconds := int(rateLimitResult.RetryAfter.Seconds())
		logger.Warn("rate limit exceeded",
			"ownerId", req.OwnerID,
			"location", req.Location,
			"retryAfter", retryAfterSeconds,
		)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusTooManyRequests,
			Description: domain.ErrRateLimitExceeded.Error(),
			RetryAfter:  retryAfterSeconds,
		}
	}

	// Update Route53 DNS record
	if err := dnsService.UpsertRecord(ctx, subdomain, ip); err != nil {
		logger.Error("failed to update DNS record", "error", err)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to update DNS record",
		}
	}

	// Prepare mapping for storage
	var mapping domain.IPMapping
	if existing != nil {
		mapping = *existing
	} else {
		mapping = domain.IPMapping{
			OwnerID:      req.OwnerID,
			LocationName: req.Location,
			Subdomain:    subdomain,
		}
	}

	// Update IP and timestamps
	mapping.IP = ip
	mapping.UpdatedAt = now
	// Only set subdomain for new mappings; preserve existing custom subdomains
	if isNew {
		mapping.Subdomain = subdomain
	}

	// Update rate limit counters
	ratelimit.UpdateCounters(&mapping, now)

	// Save to repository
	if err := repo.Put(ctx, mapping); err != nil {
		logger.Error("failed to save mapping", "error", err)
		return response.MappingResponse{}, &response.RequestError{
			Status:      http.StatusInternalServerError,
			Description: "failed to save mapping",
		}
	}

	logger.Info("IP updated successfully",
		"ownerId", mapping.OwnerID,
		"location", mapping.LocationName,
		"ip", mapping.IP,
		"subdomain", fullSubdomain,
		"isNew", isNew,
	)

	return response.MappingResponse{
		Status: http.StatusOK,
		Body: response.MappingBody{
			OwnerID:   mapping.OwnerID,
			Location:  mapping.LocationName,
			IP:        mapping.IP,
			Subdomain: fullSubdomain,
			Changed:   true,
			UpdatedAt: mapping.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

// extractClientIP gets the client IP from the request.
// It checks X-Forwarded-For header first, then falls back to SourceIP.
func extractClientIP(request events.APIGatewayProxyRequest) string {
	// Check X-Forwarded-For header (set by API Gateway/load balancers)
	xff := request.Headers["X-Forwarded-For"]
	if xff == "" {
		xff = request.Headers["x-forwarded-for"]
	}
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Fall back to SourceIP from request context
	if request.RequestContext.Identity.SourceIP != "" {
		return request.RequestContext.Identity.SourceIP
	}

	return ""
}
