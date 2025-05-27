// ABOUTME: Share service handles URL sharing functionality
// ABOUTME: Provides business logic for creating and retrieving shared URL collections

package share

import (
	"context"
	"errors"

	"digests-app-api/core/domain"
	"digests-app-api/core/interfaces"
	"github.com/google/uuid"
)

// ShareService handles share operations
type ShareService struct {
	storage interfaces.ShareStorage
}

// NewShareService creates a new share service instance
func NewShareService(storage interfaces.ShareStorage) *ShareService {
	return &ShareService{
		storage: storage,
	}
}

// CreateShare creates a new share with the given URLs
func (s *ShareService) CreateShare(ctx context.Context, urls []string) (*domain.Share, error) {
	// Use domain model's validation
	share, err := domain.NewShare(urls)
	if err != nil {
		return nil, err
	}

	// Save to storage
	if err := s.storage.Save(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// GetShare retrieves a share by ID
func (s *ShareService) GetShare(ctx context.Context, id string) (*domain.Share, error) {
	if id == "" {
		return nil, errors.New("share ID cannot be empty")
	}

	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("invalid share ID format")
	}

	// Get from storage
	share, err := s.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if share exists
	if share == nil {
		return nil, nil
	}

	// Check if share is expired
	if share.IsExpired() {
		return nil, errors.New("share has expired")
	}

	return share, nil
}