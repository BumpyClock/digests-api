// ABOUTME: Response DTOs for reader view API endpoints
// ABOUTME: Defines the structure for reader view extraction responses

package responses

import "digests-app-api/core/domain"

// ReaderViewResponse represents the response for reader view extraction
type ReaderViewResponse struct {
	Views []domain.ReaderView `json:"views"`
}