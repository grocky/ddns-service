package domain

import (
	"strings"
	"time"
)

// Owner represents a registered owner in the system.
type Owner struct {
	OwnerID    string    `dynamodbav:"OwnerId"`
	Email      string    `dynamodbav:"Email"`
	APIKeyHash string    `dynamodbav:"ApiKeyHash"`
	CreatedAt  time.Time `dynamodbav:"CreatedAt"`
}

// CreateOwnerRequest represents a request to create a new owner.
type CreateOwnerRequest struct {
	OwnerID string `json:"ownerId"`
	Email   string `json:"email"`
}

// Validate checks that the request has all required fields.
func (r CreateOwnerRequest) Validate() error {
	if strings.TrimSpace(r.OwnerID) == "" {
		return ErrMissingOwnerID
	}
	if strings.TrimSpace(r.Email) == "" {
		return ErrMissingEmail
	}
	if !isValidEmail(r.Email) {
		return ErrInvalidEmail
	}
	return nil
}

// RecoverKeyRequest represents a request to recover an API key.
type RecoverKeyRequest struct {
	Email string `json:"email"`
}

// Validate checks that the request has all required fields.
func (r RecoverKeyRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return ErrMissingEmail
	}
	return nil
}

// isValidEmail performs basic email validation.
func isValidEmail(email string) bool {
	// Basic check: contains @ and has something before and after
	at := strings.Index(email, "@")
	if at < 1 {
		return false
	}
	dot := strings.LastIndex(email, ".")
	return dot > at+1 && dot < len(email)-1
}
