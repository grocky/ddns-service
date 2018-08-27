package main

import (
  "encoding/json"
  "fmt"
  "github.com/aws/aws-lambda-go/events"
  "github.com/aws/aws-lambda-go/lambda"
  "log"
  "net/http"
  "os"
)

var errorLogger = log.New(os.Stderr, "ERROR: ", log.Llongfile)

type ClientIpResponse struct {
  PublicIp string `json:"publicIp"`
}

type ErrorResponse struct {
  Description string `json:"description"`
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

  clientIp := request.Headers["X-Forwarded-For"]

  if clientIp == "" {
    return clientError(http.StatusBadRequest, "Client IP not found");
  }

  js, err := json.Marshal(ClientIpResponse{clientIp})
  if err != nil {
   return serverError(err)
  }

  return events.APIGatewayProxyResponse{
    StatusCode: 200,
    Body: string(js),
  }, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
  errorLogger.Println(err.Error())

  return events.APIGatewayProxyResponse{
    StatusCode: http.StatusInternalServerError,
    Body:       buildErrorResponse(err.Error()),
  }, nil
}

// Similarly add a helper for send responses relating to client errors.
func clientError(status int, description string) (events.APIGatewayProxyResponse, error) {
  return events.APIGatewayProxyResponse{
    StatusCode: status,
    Body: buildErrorResponse(description),
  }, nil
}

func buildErrorResponse(description string) string {
  response := ErrorResponse{Description: description}
  js, err := json.Marshal(response)

  if (err != nil) {
    errorLogger.Println(response)
    return "Unable to marshl response."
  }

  return string(js)
}

func main() {
  fmt.Print(os.Getenv("_LAMBDA_SERVER_PORT"))
  lambda.Start(Handler)
}
