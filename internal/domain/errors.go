package domain

import "errors"

var (
	// ErrMissingOwnerID is returned when ownerId is empty.
	ErrMissingOwnerID = errors.New("ownerId is required")

	// ErrMissingLocation is returned when location is empty.
	ErrMissingLocation = errors.New("location is required")

	// ErrMappingNotFound is returned when a mapping doesn't exist.
	ErrMappingNotFound = errors.New("mapping not found")

	// ErrInvalidIP is returned when the IP address is invalid.
	ErrInvalidIP = errors.New("invalid IP address")
)
