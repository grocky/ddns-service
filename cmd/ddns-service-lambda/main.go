package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/grocky/ddns-service/internal/handlers"
	"github.com/grocky/ddns-service/internal/response"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// Handler handles API Gateway proxy requests.
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	method := request.HTTPMethod
	route := request.Path

	logger.Info("request received", "method", method, "route", route)

	if method == http.MethodGet && route == "/public-ip" {
		resp, reqErr := handlers.GetPublicIP(request, logger)
		if reqErr != nil {
			return clientError(reqErr)
		}

		js, err := json.Marshal(resp.Body)
		if err != nil {
			return serverError(err)
		}

		return events.APIGatewayProxyResponse{
			StatusCode: resp.Status,
			Body:       string(js),
		}, nil
	}

	logger.Warn("resource not found", "route", route)
	return clientError(&response.RequestError{
		Status:      http.StatusNotFound,
		Description: fmt.Sprintf("Resource not found: %s", route),
	})
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	logger.Error("server error", "error", err)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       response.BuildErrorJSON(err.Error(), logger),
	}, nil
}

func clientError(reqErr *response.RequestError) (events.APIGatewayProxyResponse, error) {
	logger.Debug("client error", "status", reqErr.Status, "description", reqErr.Description)

	return events.APIGatewayProxyResponse{
		StatusCode: reqErr.Status,
		Body:       response.BuildErrorJSON(reqErr.Error(), logger),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
