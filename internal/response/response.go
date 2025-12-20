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

// MappingResponse represents a response containing an IP mapping.
type MappingResponse struct {
	Status int
	Body   MappingBody
}

// MappingBody is the JSON body for a mapping response.
type MappingBody struct {
	OwnerID   string `json:"ownerId"`
	Location  string `json:"location"`
	IP        string `json:"ip"`
	Subdomain string `json:"subdomain"`
	Changed   bool   `json:"changed,omitempty"`
	UpdatedAt string `json:"updatedAt"`
}

// OwnerResponse represents a response containing owner information.
type OwnerResponse struct {
	Status int
	Body   OwnerBody
}

// OwnerBody is the JSON body for an owner response.
type OwnerBody struct {
	OwnerID   string `json:"ownerId"`
	Email     string `json:"email,omitempty"`
	APIKey    string `json:"apiKey,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	RotatedAt string `json:"rotatedAt,omitempty"`
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Status int
	Body   MessageBody
}

// MessageBody is the JSON body for a message response.
type MessageBody struct {
	Message string `json:"message"`
}

// ACMEChallengeResponse represents a response for creating an ACME challenge.
type ACMEChallengeResponse struct {
	Status int
	Body   ACMEChallengeBody
}

// ACMEChallengeBody is the JSON body for an ACME challenge response.
type ACMEChallengeBody struct {
	OwnerID   string `json:"ownerId"`
	Location  string `json:"location"`
	Subdomain string `json:"subdomain"`
	TxtRecord string `json:"txtRecord"`
	TxtValue  string `json:"txtValue"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

// ACMEDeleteResponse represents a response for deleting an ACME challenge.
type ACMEDeleteResponse struct {
	Status int
	Body   ACMEDeleteBody
}

// ACMEDeleteBody is the JSON body for an ACME delete response.
type ACMEDeleteBody struct {
	OwnerID   string `json:"ownerId"`
	Location  string `json:"location"`
	Subdomain string `json:"subdomain"`
	TxtRecord string `json:"txtRecord"`
	Deleted   bool   `json:"deleted"`
}

// ErrorBody is the JSON body for error responses.
type ErrorBody struct {
	Description string `json:"description"`
}

// RequestError represents an error with an associated HTTP status code.
type RequestError struct {
	Status      int
	Description string
	RetryAfter  int // Seconds until retry is allowed (for rate limiting)
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
