package handlers

import (
  "github.com/aws/aws-lambda-go/events"
  "log"
  "net/http"
)

type ResponseBody struct {
  PublicIp string `json:"publicIp"`
}

type ClientIpResponse struct {
  Status int
  Body ResponseBody
}

type ErrorResponse struct {
  Description string `json:"description"`
}

type RequestError struct {
  Status      int
  Description string
}

func (e *RequestError) Error() string {
  return e.Description
}

func GetPublicIPHandler(request events.APIGatewayProxyRequest, logger log.Logger) (ClientIpResponse, *RequestError) {

  var requestError *RequestError
  var response ClientIpResponse

  logger.Println("GetPublicIPHandler : started")
  defer logger.Println("GetPublicIPHandler : completed")

  clientIp := request.Headers["X-Forwarded-For"]

  if clientIp == "" {
    return response, &RequestError{http.StatusBadRequest, "Client IP not found"}
  }

  logger.Printf("Recognized public ip: %s", clientIp)

  return ClientIpResponse{http.StatusOK, ResponseBody{PublicIp: clientIp}}, requestError
}

