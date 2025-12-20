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
