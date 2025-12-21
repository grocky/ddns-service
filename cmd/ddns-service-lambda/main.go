package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/grocky/ddns-service/internal/dns"
	"github.com/grocky/ddns-service/internal/email"
	"github.com/grocky/ddns-service/internal/handlers"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

var (
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	repo     repository.Repository
	emailSvc email.Service
	dnsSvc   dns.Service
	initOnce sync.Once
	initErr  error
)

func initServices(ctx context.Context) error {
	initOnce.Do(func() {
		logger.Info("initializing services")

		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			logger.Error("failed to load AWS config", "error", err)
			initErr = err
			return
		}

		// Initialize DynamoDB repository
		dynamoClient := dynamodb.NewFromConfig(cfg)
		repo = repository.NewDynamoDBRepository(dynamoClient, logger)

		// Initialize SES email service
		sesClient := ses.NewFromConfig(cfg)
		emailSvc = email.NewSESService(sesClient, logger)

		// Initialize Route53 DNS service
		hostedZoneID := os.Getenv("ROUTE53_HOSTED_ZONE_ID")
		if hostedZoneID == "" {
			logger.Warn("ROUTE53_HOSTED_ZONE_ID not set, DNS updates will fail")
		}
		route53Client := route53.NewFromConfig(cfg)
		dnsSvc = dns.NewRoute53Service(route53Client, hostedZoneID, logger)

		logger.Info("services initialized")
	})
	return initErr
}

// EventBridgeEvent represents an EventBridge scheduled event.
type EventBridgeEvent struct {
	Source string `json:"source"`
	Action string `json:"action"`
}

// GenericHandler handles both API Gateway and EventBridge events.
func GenericHandler(ctx context.Context, rawEvent json.RawMessage) (any, error) {
	// Try to detect if this is an EventBridge event
	var ebEvent EventBridgeEvent
	if err := json.Unmarshal(rawEvent, &ebEvent); err == nil && ebEvent.Source == "ddns.acme-cleanup" {
		return handleEventBridge(ctx, ebEvent)
	}

	// Otherwise, treat as API Gateway request
	var request events.APIGatewayProxyRequest
	if err := json.Unmarshal(rawEvent, &request); err != nil {
		logger.Error("failed to unmarshal event", "error", err)
		return nil, err
	}

	return Handler(ctx, request)
}

// handleEventBridge processes EventBridge scheduled events.
func handleEventBridge(ctx context.Context, event EventBridgeEvent) (any, error) {
	logger.Info("EventBridge event received", "source", event.Source, "action", event.Action)

	if err := initServices(ctx); err != nil {
		logger.Error("failed to initialize services", "error", err)
		return nil, err
	}

	switch event.Action {
	case "cleanup-expired-challenges":
		return handlers.CleanupExpiredChallenges(ctx, repo, dnsSvc, logger)
	default:
		logger.Warn("unknown EventBridge action", "action", event.Action)
		return nil, fmt.Errorf("unknown action: %s", event.Action)
	}
}

// Handler handles API Gateway proxy requests.
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	method := request.HTTPMethod
	route := request.Path

	logger.Info("request received", "method", method, "route", route)

	// Initialize services for routes that need them
	if err := initServices(ctx); err != nil {
		return serverError(fmt.Errorf("failed to initialize: %w", err))
	}

	// POST /owners - create new owner
	if method == http.MethodPost && route == "/owners" {
		resp, reqErr := handlers.CreateOwner(ctx, request, repo, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// POST /owners/{ownerId}/recover - recover API key
	if method == http.MethodPost && strings.HasPrefix(route, "/owners/") && strings.HasSuffix(route, "/recover") {
		ownerID := extractOwnerIDFromPath(route, "/owners/", "/recover")
		if ownerID == "" {
			return clientError(&response.RequestError{
				Status:      http.StatusBadRequest,
				Description: "invalid owner path",
			})
		}
		resp, reqErr := handlers.RecoverKey(ctx, request, ownerID, repo, emailSvc, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// POST /owners/{ownerId}/rotate - rotate API key
	if method == http.MethodPost && strings.HasPrefix(route, "/owners/") && strings.HasSuffix(route, "/rotate") {
		ownerID := extractOwnerIDFromPath(route, "/owners/", "/rotate")
		if ownerID == "" {
			return clientError(&response.RequestError{
				Status:      http.StatusBadRequest,
				Description: "invalid owner path",
			})
		}
		resp, reqErr := handlers.RotateKey(ctx, request, ownerID, repo, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// POST /register - register IP (requires auth) - deprecated, use /update
	if method == http.MethodPost && route == "/register" {
		resp, reqErr := handlers.Register(ctx, request, repo, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// POST /update - update DNS if IP changed (requires auth)
	if method == http.MethodPost && route == "/update" {
		resp, reqErr := handlers.Update(ctx, request, repo, dnsSvc, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// GET /lookup/{ownerId}/{location} - lookup IP (requires auth)
	if method == http.MethodGet && strings.HasPrefix(route, "/lookup/") {
		resp, reqErr := handlers.Lookup(ctx, request, repo, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// POST /acme-challenge - create ACME challenge TXT record (requires auth)
	if method == http.MethodPost && route == "/acme-challenge" {
		resp, reqErr := handlers.CreateACMEChallenge(ctx, request, repo, dnsSvc, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// DELETE /acme-challenge - delete ACME challenge TXT record (requires auth)
	if method == http.MethodDelete && route == "/acme-challenge" {
		resp, reqErr := handlers.DeleteACMEChallenge(ctx, request, repo, dnsSvc, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	logger.Warn("resource not found", "route", route)
	return clientError(&response.RequestError{
		Status:      http.StatusNotFound,
		Description: fmt.Sprintf("Resource not found: %s", route),
	})
}

// extractOwnerIDFromPath extracts the owner ID from paths like /owners/{ownerId}/action
func extractOwnerIDFromPath(path, prefix, suffix string) string {
	path = strings.TrimPrefix(path, prefix)
	path = strings.TrimSuffix(path, suffix)
	return path
}

func jsonResponse(status int, body any) (events.APIGatewayProxyResponse, error) {
	js, err := json.Marshal(body)
	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(js),
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	logger.Error("server error", "error", err)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: response.BuildErrorJSON(err.Error(), logger),
	}, nil
}

func clientError(reqErr *response.RequestError) (events.APIGatewayProxyResponse, error) {
	logger.Debug("client error", "status", reqErr.Status, "description", reqErr.Description)

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Add Retry-After header for rate limiting
	if reqErr.RetryAfter > 0 {
		headers["Retry-After"] = strconv.Itoa(reqErr.RetryAfter)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: reqErr.Status,
		Headers:    headers,
		Body:       response.BuildErrorJSON(reqErr.Error(), logger),
	}, nil
}

func main() {
	lambda.Start(GenericHandler)
}
