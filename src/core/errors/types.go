// ABOUTME: Custom error types for the core business logic
// ABOUTME: Provides structured errors for better error handling and API responses

package errors

import (
	"errors"
	"fmt"
)

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Resource string
	ID       string
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ExternalAPIError represents an error from an external API
type ExternalAPIError struct {
	StatusCode int
	Message    string
	API        string
}

// Error implements the error interface
func (e *ExternalAPIError) Error() string {
	return fmt.Sprintf("external API error from %s: %d - %s", e.API, e.StatusCode, e.Message)
}

// IsNotFound checks if an error is a NotFoundError
func IsNotFound(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// IsValidation checks if an error is a ValidationError
func IsValidation(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsExternalAPI checks if an error is an ExternalAPIError
func IsExternalAPI(err error) bool {
	var apiErr *ExternalAPIError
	return errors.As(err, &apiErr)
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}