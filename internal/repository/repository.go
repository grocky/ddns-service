package repository

import (
	"context"
	"errors"

	"github.com/grocky/ddns-service/internal/domain"
)

// Repository defines the interface for IP mapping and owner storage.
type Repository interface {
	// Put creates or updates an IP mapping.
	Put(ctx context.Context, mapping domain.IPMapping) error

	// Get retrieves an IP mapping by owner ID and location.
	Get(ctx context.Context, ownerID, location string) (*domain.IPMapping, error)

	// CreateOwner creates a new owner. Returns ErrOwnerExists if owner already exists.
	CreateOwner(ctx context.Context, owner domain.Owner) error

	// GetOwner retrieves an owner by ID. Returns ErrOwnerNotFound if not found.
	GetOwner(ctx context.Context, ownerID string) (*domain.Owner, error)

	// UpdateOwnerKey updates the API key hash for an owner.
	UpdateOwnerKey(ctx context.Context, ownerID, newKeyHash string) error
}

// IsOwnerNotFound returns true if the error is ErrOwnerNotFound.
func IsOwnerNotFound(err error) bool {
	return errors.Is(err, domain.ErrOwnerNotFound)
}

// IsOwnerExists returns true if the error is ErrOwnerExists.
func IsOwnerExists(err error) bool {
	return errors.Is(err, domain.ErrOwnerExists)
}

// IsMappingNotFound returns true if the error is ErrMappingNotFound.
func IsMappingNotFound(err error) bool {
	return errors.Is(err, domain.ErrMappingNotFound)
}
