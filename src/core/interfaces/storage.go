// ABOUTME: Storage interfaces for persisting domain entities
// ABOUTME: Defines contracts for data persistence operations

package interfaces

import (
	"context"
	
	"digests-app-api/core/domain"
)

// ShareStorage defines the interface for share persistence
type ShareStorage interface {
	// Save persists a share
	Save(ctx context.Context, share *domain.Share) error

	// Get retrieves a share by ID
	Get(ctx context.Context, id string) (*domain.Share, error)
}