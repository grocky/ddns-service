package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
  "github.com/grocky/ddns-service/cmd/ddns-service-lambda/handlers"
  "log"
	"net/http"
	"os"
)

var logger = log.New(os.Stdout, "ddns-service : ", log.LstdFlags | log.Llongfile )

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	method := request.HTTPMethod
	route := request.Path

	var response handlers.ClientIpResponse
	var requestError *handlers.RequestError

	if method == http.MethodGet && route == "/public-ip" {
		response, requestError = handlers.GetPublicIPHandler(request, *logger)
    if requestError != nil {
      return clientError(*requestError)
    }

    js, err := json.Marshal(response.Body)
    if err != nil {
      return serverError(err)
    }

    return events.APIGatewayProxyResponse{
      StatusCode: response.Status,
      Body:       string(js),
    }, nil
	}

	return clientError(handlers.RequestError{http.StatusNotFound, fmt.Sprintf("Resource not found: %s", route)})
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	logger.Println(err.Error())

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       buildErrorResponse(err.Error()),
	}, nil
}

// Similarly add a helper for send responses relating to client errors.
func clientError(requestError handlers.RequestError) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: requestError.Status,
		Body:       buildErrorResponse(requestError.Error()),
	}, nil
}

func buildErrorResponse(description string) string {
	response := handlers.ErrorResponse{Description: description}
	js, err := json.Marshal(response)

	if err != nil {
		logger.Println(response)
		return "Unable to marshl response."
	}

	return string(js)
}

func main() {
	fmt.Print(os.Getenv("_LAMBDA_SERVER_PORT"))
	lambda.Start(Handler)
}