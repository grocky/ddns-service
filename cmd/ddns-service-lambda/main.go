package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/grocky/ddns-service/internal/handlers"
	"github.com/grocky/ddns-service/internal/repository"
	"github.com/grocky/ddns-service/internal/response"
)

var (
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	repo     repository.Repository
	initOnce sync.Once
	initErr  error
)

func initRepo(ctx context.Context) error {
	initOnce.Do(func() {
		logger.Info("initializing repository")

		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			logger.Error("failed to load AWS config", "error", err)
			initErr = err
			return
		}

		client := dynamodb.NewFromConfig(cfg)
		repo = repository.NewDynamoDBRepository(client, logger)

		logger.Info("repository initialized")
	})
	return initErr
}

// Handler handles API Gateway proxy requests.
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	method := request.HTTPMethod
	route := request.Path

	logger.Info("request received", "method", method, "route", route)

	// GET /public-ip - doesn't need DynamoDB
	if method == http.MethodGet && route == "/public-ip" {
		resp, reqErr := handlers.GetPublicIP(request, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// Initialize repository for routes that need it
	if err := initRepo(ctx); err != nil {
		return serverError(fmt.Errorf("failed to initialize: %w", err))
	}

	// POST /register
	if method == http.MethodPost && route == "/register" {
		resp, reqErr := handlers.Register(ctx, request, repo, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}
		return jsonResponse(resp.Status, resp.Body)
	}

	// GET /lookup/{ownerId}/{location}
	if method == http.MethodGet && strings.HasPrefix(route, "/lookup/") {
		resp, reqErr := handlers.Lookup(ctx, request, repo, logger)
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

	return events.APIGatewayProxyResponse{
		StatusCode: reqErr.Status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: response.BuildErrorJSON(reqErr.Error(), logger),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
