package handlers

import (
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/response"
)

// GetPublicIP extracts and returns the client's public IP from the request.
// Uses RequestContext.Identity.SourceIP as the primary source (set by API Gateway,
// cannot be spoofed), with X-Forwarded-For as a fallback.
func GetPublicIP(request events.APIGatewayProxyRequest, logger *slog.Logger) (response.ClientIPResponse, *response.RequestError) {
	logger.Info("handler started", "handler", "GetPublicIP")
	defer logger.Info("handler completed", "handler", "GetPublicIP")

	// Primary: Use SourceIP from API Gateway (authoritative, cannot be spoofed)
	clientIP := request.RequestContext.Identity.SourceIP

	// Fallback: X-Forwarded-For header
	if clientIP == "" {
		clientIP = request.Headers["X-Forwarded-For"]
	}

	if clientIP == "" {
		logger.Warn("client IP not found", "handler", "GetPublicIP")
		return response.ClientIPResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "Client IP not found",
		}
	}

	logger.Info("recognized public IP", "handler", "GetPublicIP", "clientIP", clientIP)

	return response.ClientIPResponse{
		Status: http.StatusOK,
		Body:   response.ClientIPBody{PublicIP: clientIP},
	}, nil
}
