package domain

import (
	"time"
)

// IPMapping represents a mapping between an owner's location and their IP address.
type IPMapping struct {
	OwnerID      string    `dynamodbav:"OwnerId"`
	LocationName string    `dynamodbav:"LocationName"`
	IP           string    `dynamodbav:"IP"`
	UpdatedAt    time.Time `dynamodbav:"UpdatedAt"`
}

// RegisterRequest is the request body for registering an IP mapping.
type RegisterRequest struct {
	OwnerID  string `json:"ownerId"`
	Location string `json:"location"`
	IP       string `json:"ip"` // "auto" to use X-Forwarded-For, or explicit IP
}

// Validate checks if the register request is valid.
func (r RegisterRequest) Validate() error {
	if r.OwnerID == "" {
		return ErrMissingOwnerID
	}
	if r.Location == "" {
		return ErrMissingLocation
	}
	return nil
}

