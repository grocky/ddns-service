package domain

import (
	"time"
)

// IPMapping represents a mapping between an owner's location and their IP address.
type IPMapping struct {
	OwnerID           string    `dynamodbav:"OwnerId"`
	LocationName      string    `dynamodbav:"LocationName"`
	IP                string    `dynamodbav:"IP"`
	Subdomain         string    `dynamodbav:"Subdomain"`
	UpdatedAt         time.Time `dynamodbav:"UpdatedAt"`
	LastIPChangeAt    time.Time `dynamodbav:"LastIPChangeAt"`
	HourlyChangeCount int       `dynamodbav:"HourlyChangeCount"`
}

// UpdateRequest is the request body for updating a DNS mapping.
// IP is detected automatically from the request context.
type UpdateRequest struct {
	OwnerID  string `json:"ownerId"`
	Location string `json:"location"`
}

// Validate checks if the update request is valid.
func (r UpdateRequest) Validate() error {
	if r.OwnerID == "" {
		return ErrMissingOwnerID
	}
	if r.Location == "" {
		return ErrMissingLocation
	}
	return nil
}

// RegisterRequest is deprecated, use UpdateRequest instead.
// Kept for backward compatibility.
type RegisterRequest = UpdateRequest

