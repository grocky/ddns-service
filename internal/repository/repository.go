package repository

import (
	"context"

	"github.com/grocky/ddns-service/internal/domain"
)

// Repository defines the interface for IP mapping storage.
type Repository interface {
	// Put creates or updates an IP mapping.
	Put(ctx context.Context, mapping domain.IPMapping) error

	// Get retrieves an IP mapping by owner ID and location.
	Get(ctx context.Context, ownerID, location string) (*domain.IPMapping, error)
}
