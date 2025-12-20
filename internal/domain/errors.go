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

	// ErrRateLimitExceeded is returned when too many IP changes occur in an hour.
	ErrRateLimitExceeded = errors.New("rate limit exceeded: maximum 2 IP changes per hour")

	// ErrMissingIP is returned when the client IP cannot be determined.
	ErrMissingIP = errors.New("could not determine client IP")

	// ErrMissingTxtValue is returned when txtValue is empty.
	ErrMissingTxtValue = errors.New("txtValue is required")

	// ErrInvalidTxtValue is returned when txtValue format is invalid.
	ErrInvalidTxtValue = errors.New("invalid txtValue format")

	// ErrChallengeNotFound is returned when an ACME challenge doesn't exist.
	ErrChallengeNotFound = errors.New("challenge not found")

	// ErrChallengeExists is returned when an ACME challenge already exists.
	ErrChallengeExists = errors.New("challenge already exists")

	// ErrACMERateLimitExceeded is returned when too many ACME challenges are created.
	ErrACMERateLimitExceeded = errors.New("rate limit exceeded: maximum 10 challenges per hour")
)
