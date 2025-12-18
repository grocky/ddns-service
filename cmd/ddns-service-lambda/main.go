package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/grocky/ddns-service/internal/handlers"
	"github.com/grocky/ddns-service/internal/response"
)

var logger = log.New(os.Stdout, "ddns-service: ", log.LstdFlags|log.Lshortfile)

// Handler handles API Gateway proxy requests.
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	method := request.HTTPMethod
	route := request.Path

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

	return clientError(&response.RequestError{
		Status:      http.StatusNotFound,
		Description: fmt.Sprintf("Resource not found: %s", route),
	})
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	logger.Printf("server error: %v", err)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       response.BuildErrorJSON(err.Error(), logger),
	}, nil
}

func clientError(reqErr *response.RequestError) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: reqErr.Status,
		Body:       response.BuildErrorJSON(reqErr.Error(), logger),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
