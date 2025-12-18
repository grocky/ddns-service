package handlers

import (
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grocky/ddns-service/internal/response"
)

// GetPublicIP extracts and returns the client's public IP from the request headers.
func GetPublicIP(request events.APIGatewayProxyRequest, logger *log.Logger) (response.ClientIPResponse, *response.RequestError) {
	logger.Println("GetPublicIP: started")
	defer logger.Println("GetPublicIP: completed")

	clientIP := request.Headers["X-Forwarded-For"]
	if clientIP == "" {
		return response.ClientIPResponse{}, &response.RequestError{
			Status:      http.StatusBadRequest,
			Description: "Client IP not found",
		}
	}

	logger.Printf("GetPublicIP: recognized public IP: %s", clientIP)

	return response.ClientIPResponse{
		Status: http.StatusOK,
		Body:   response.ClientIPBody{PublicIP: clientIP},
	}, nil
}
