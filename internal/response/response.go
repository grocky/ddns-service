package response

import (
	"encoding/json"
	"log/slog"
)

// ClientIPResponse represents a successful response containing the client's IP.
type ClientIPResponse struct {
	Status int
	Body   ClientIPBody
}

// ClientIPBody is the JSON body for a client IP response.
type ClientIPBody struct {
	PublicIP string `json:"publicIp"`
}

// ErrorBody is the JSON body for error responses.
type ErrorBody struct {
	Description string `json:"description"`
}

// RequestError represents an error with an associated HTTP status code.
type RequestError struct {
	Status      int
	Description string
}

// Error implements the error interface.
func (e *RequestError) Error() string {
	return e.Description
}

// BuildErrorJSON marshals an error description to JSON.
// If marshaling fails, it logs the error and returns a fallback message.
func BuildErrorJSON(description string, logger *slog.Logger) string {
	body := ErrorBody{Description: description}
	js, err := json.Marshal(body)
	if err != nil {
		logger.Error("failed to marshal error response", "error", err)
		return `{"description":"internal server error"}`
	}
	return string(js)
}
