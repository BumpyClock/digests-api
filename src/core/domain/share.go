// ABOUTME: Share domain model represents a collection of URLs shared with an expiration
// ABOUTME: Provides validation and expiration checking for shared URL collections

package domain

import (
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// Share represents a shareable collection of URLs
type Share struct {
	// ID is the unique identifier (UUID) for the share
	ID string

	// URLs contains the list of shared URLs
	URLs []string

	// CreatedAt is when the share was created
	CreatedAt time.Time

	// ExpiresAt is when the share expires (nil means no expiration)
	ExpiresAt *time.Time
}

// NewShare creates a new Share instance with validation
func NewShare(urls []string) (*Share, error) {
	if len(urls) == 0 {
		return nil, errors.New("urls cannot be empty")
	}

	// Validate each URL
	for _, u := range urls {
		parsedURL, err := url.Parse(u)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			return nil, errors.New("each URL must be valid")
		}
	}

	share := &Share{
		ID:        uuid.New().String(),
		URLs:      urls,
		CreatedAt: time.Now(),
		ExpiresAt: nil,
	}

	return share, nil
}

// IsExpired checks if the share has expired
func (s *Share) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}

	return time.Now().After(*s.ExpiresAt)
}