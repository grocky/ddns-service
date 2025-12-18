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

	// ErrMissingEmail is returned when email is empty.
	ErrMissingEmail = errors.New("email is required")

	// ErrInvalidEmail is returned when email format is invalid.
	ErrInvalidEmail = errors.New("invalid email address")

	// ErrOwnerExists is returned when trying to create an owner that already exists.
	ErrOwnerExists = errors.New("owner already exists")

	// ErrOwnerNotFound is returned when an owner doesn't exist.
	ErrOwnerNotFound = errors.New("owner not found")

	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the authenticated owner doesn't match the requested resource.
	ErrForbidden = errors.New("forbidden")
)
