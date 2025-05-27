package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{
		Resource: "feed",
		ID:       "123",
	}
	
	expected := "feed not found: 123"
	if err.Error() != expected {
		t.Errorf("NotFoundError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "email",
		Message: "invalid email format",
	}
	
	expected := "validation error on field 'email': invalid email format"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestExternalAPIError_Error(t *testing.T) {
	err := &ExternalAPIError{
		StatusCode: 503,
		Message:    "service unavailable",
		API:        "feedsearch",
	}
	
	expected := "external API error from feedsearch: 503 - service unavailable"
	if err.Error() != expected {
		t.Errorf("ExternalAPIError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestIsNotFound_True(t *testing.T) {
	err := &NotFoundError{
		Resource: "share",
		ID:       "abc",
	}
	
	if !IsNotFound(err) {
		t.Error("IsNotFound should return true for NotFoundError")
	}
}

func TestIsNotFound_False(t *testing.T) {
	err := errors.New("some other error")
	
	if IsNotFound(err) {
		t.Error("IsNotFound should return false for non-NotFoundError")
	}
}

func TestIsNotFound_WrappedError(t *testing.T) {
	notFound := &NotFoundError{
		Resource: "feed",
		ID:       "123",
	}
	wrapped := fmt.Errorf("failed to get feed: %w", notFound)
	
	if !IsNotFound(wrapped) {
		t.Error("IsNotFound should return true for wrapped NotFoundError")
	}
}

func TestIsValidation_True(t *testing.T) {
	err := &ValidationError{
		Field:   "url",
		Message: "invalid URL",
	}
	
	if !IsValidation(err) {
		t.Error("IsValidation should return true for ValidationError")
	}
}

func TestIsValidation_False(t *testing.T) {
	err := errors.New("some other error")
	
	if IsValidation(err) {
		t.Error("IsValidation should return false for non-ValidationError")
	}
}

func TestIsExternalAPI_True(t *testing.T) {
	err := &ExternalAPIError{
		StatusCode: 500,
		Message:    "internal server error",
		API:        "search",
	}
	
	if !IsExternalAPI(err) {
		t.Error("IsExternalAPI should return true for ExternalAPIError")
	}
}

func TestIsExternalAPI_False(t *testing.T) {
	err := errors.New("some other error")
	
	if IsExternalAPI(err) {
		t.Error("IsExternalAPI should return false for non-ExternalAPIError")
	}
}

func TestWrapError_PreservesOriginalError(t *testing.T) {
	originalErr := &NotFoundError{Resource: "feed", ID: "abc"}
	wrappedErr := WrapError(originalErr, "failed to fetch feed")
	
	if wrappedErr == nil {
		t.Fatal("WrapError should not return nil for non-nil error")
	}
	
	// Check error message contains both context and original error
	expectedMsg := "failed to fetch feed: feed not found: abc"
	if wrappedErr.Error() != expectedMsg {
		t.Errorf("WrapError message = %v, want %v", wrappedErr.Error(), expectedMsg)
	}
	
	// Should still be identifiable as NotFoundError
	if !IsNotFound(wrappedErr) {
		t.Error("Wrapped error should still be identifiable as NotFoundError")
	}
}

func TestWrapError_AddsContextMessage(t *testing.T) {
	originalErr := errors.New("network timeout")
	wrappedErr := WrapError(originalErr, "external API call failed")
	
	expected := "external API call failed: network timeout"
	if wrappedErr.Error() != expected {
		t.Errorf("WrapError = %v, want %v", wrappedErr.Error(), expected)
	}
}

func TestWrapError_HandlesNilError(t *testing.T) {
	wrappedErr := WrapError(nil, "this should not happen")
	
	if wrappedErr != nil {
		t.Error("WrapError should return nil when wrapping nil error")
	}
}