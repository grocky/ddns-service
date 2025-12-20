package domain

import (
	"time"
)

// ACMEChallenge represents an active ACME DNS-01 challenge.
type ACMEChallenge struct {
	OwnerID      string    `dynamodbav:"OwnerId" json:"ownerId"`
	LocationName string    `dynamodbav:"LocationName" json:"location"`
	Subdomain    string    `dynamodbav:"Subdomain" json:"subdomain"`
	TxtValue     string    `dynamodbav:"TxtValue" json:"txtValue"`
	TxtRecord    string    `dynamodbav:"TxtRecord" json:"txtRecord"`
	CreatedAt    time.Time `dynamodbav:"CreatedAt" json:"createdAt"`
	ExpiresAt    time.Time `dynamodbav:"ExpiresAt" json:"expiresAt"`
	TTL          int64     `dynamodbav:"TTL" json:"-"`
}

// CreateChallengeRequest is the request body for creating an ACME challenge.
type CreateChallengeRequest struct {
	OwnerID  string `json:"ownerId"`
	Location string `json:"location"`
	TxtValue string `json:"txtValue"`
}

// Validate checks if the request is valid.
func (r CreateChallengeRequest) Validate() error {
	if r.OwnerID == "" {
		return ErrMissingOwnerID
	}
	if r.Location == "" {
		return ErrMissingLocation
	}
	if r.TxtValue == "" {
		return ErrMissingTxtValue
	}
	// ACME tokens are base64url encoded, typically 43 chars
	if len(r.TxtValue) < 20 || len(r.TxtValue) > 100 {
		return ErrInvalidTxtValue
	}
	return nil
}

// DeleteChallengeRequest is the request body for deleting an ACME challenge.
type DeleteChallengeRequest struct {
	OwnerID  string `json:"ownerId"`
	Location string `json:"location"`
}

// Validate checks if the request is valid.
func (r DeleteChallengeRequest) Validate() error {
	if r.OwnerID == "" {
		return ErrMissingOwnerID
	}
	if r.Location == "" {
		return ErrMissingLocation
	}
	return nil
}
