// ABOUTME: Error handling utilities for API handlers
// ABOUTME: Converts domain errors to appropriate HTTP responses

package handlers

import (
	"digests-app-api/core/errors"
	"github.com/danielgtaylor/huma/v2"
)

// toHumaError converts domain errors to appropriate Huma HTTP errors
func toHumaError(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific error types
	if errors.IsNotFound(err) {
		return huma.Error404NotFound(err.Error())
	}

	if errors.IsValidation(err) {
		return huma.Error400BadRequest(err.Error())
	}

	if errors.IsExternalAPI(err) {
		// External API errors might be retryable
		if apiErr, ok := err.(*errors.ExternalAPIError); ok {
			// Map external API status codes to our API status codes
			switch {
			case apiErr.StatusCode >= 500:
				return huma.Error503ServiceUnavailable("External service error", err)
			case apiErr.StatusCode == 429:
				return huma.Error429TooManyRequests("Rate limited by external service")
			case apiErr.StatusCode >= 400:
				return huma.Error400BadRequest("External service request error", err)
			default:
				return huma.Error500InternalServerError("Unexpected external service response", err)
			}
		}
	}

	// Default to internal server error for unknown errors
	return huma.Error500InternalServerError("Internal server error", err)
}