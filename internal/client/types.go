package client

import "time"

// UpdateRequest represents the request to update DNS.
type UpdateRequest struct {
	OwnerID  string `json:"ownerId"`
	Location string `json:"location"`
	IP       string `json:"ip,omitempty"`
}

// UpdateResponse represents the server response.
type UpdateResponse struct {
	OwnerID   string `json:"ownerId"`
	Location  string `json:"location"`
	IP        string `json:"ip"`
	Subdomain string `json:"subdomain"`
	Changed   bool   `json:"changed"`
	UpdatedAt string `json:"updatedAt"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Description string `json:"description"`
}

// Config holds client configuration.
type Config struct {
	APIURL   string
	APIKey   string
	Owner    string
	Location string
	Timeout  time.Duration
}

// CreateChallengeRequest represents the request to create an ACME challenge.
type CreateChallengeRequest struct {
	OwnerID  string `json:"ownerId"`
	Location string `json:"location"`
	TxtValue string `json:"txtValue"`
}

// DeleteChallengeRequest represents the request to delete an ACME challenge.
type DeleteChallengeRequest struct {
	OwnerID  string `json:"ownerId"`
	Location string `json:"location"`
}

// ACMEChallengeResponse represents the response from creating an ACME challenge.
type ACMEChallengeResponse struct {
	OwnerID   string `json:"ownerId"`
	Location  string `json:"location"`
	Subdomain string `json:"subdomain"`
	TxtRecord string `json:"txtRecord"`
	TxtValue  string `json:"txtValue"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

// ACMEDeleteResponse represents the response from deleting an ACME challenge.
type ACMEDeleteResponse struct {
	OwnerID   string `json:"ownerId"`
	Location  string `json:"location"`
	Subdomain string `json:"subdomain"`
	TxtRecord string `json:"txtRecord"`
	Deleted   bool   `json:"deleted"`
}

// LookupResponse represents the response from looking up a mapping.
type LookupResponse struct {
	OwnerID   string `json:"ownerId"`
	Location  string `json:"location"`
	IP        string `json:"ip"`
	Subdomain string `json:"subdomain"`
	UpdatedAt string `json:"updatedAt"`
}
