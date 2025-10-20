// Package errors provides custom error types with error codes, messages, and stack traces.
package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
	"time"
)

// APIError represents an API error with error code, message, and details.
type APIError struct {
	Code      int    `json:"code"`                // HTTP status code
	Message   string `json:"message"`             // Error message
	Details   string `json:"details,omitempty"`   // Additional error details
	Timestamp string `json:"timestamp,omitempty"` // Error timestamp
	err       error  // Wrapped error (not exported to JSON)
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap implements the Unwrap method for error unwrapping compatibility.
// Returns the wrapped error if present, otherwise nil.
func (e *APIError) Unwrap() error {
	return e.err
}

// Is implements the Is method for error comparison compatibility.
// Returns true if:
// 1. e and target are the same instance (pointer equality), OR
// 2. e and target have the same error code
func (e *APIError) Is(target error) bool {
	if targetErr, ok := target.(*APIError); ok {
		// Check pointer equality first (for predefined errors)
		if e == targetErr {
			return true
		}
		// Then check code equality (for error type matching)
		return e.Code == targetErr.Code
	}
	return false
}

// As implements the As method for error type assertion compatibility.
func (e *APIError) As(target interface{}) bool {
	if targetErr, ok := target.(**APIError); ok {
		*targetErr = e
		return true
	}
	return false
}

// New creates a new APIError with the given code and message.
func New(code int, message string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Timestamp: getCurrentTimestamp(),
		err:       nil,
	}
}

// NewWithDetails creates a new APIError with code, message, and details.
func NewWithDetails(code int, message, details string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: getCurrentTimestamp(),
		err:       nil,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, code int, message string) *APIError {
	apiErr := &APIError{
		Code:      code,
		Message:   message,
		Timestamp: getCurrentTimestamp(),
		err:       err,
	}

	if err != nil {
		apiErr.Details = err.Error()
	}

	return apiErr
}

// WrapWithDetails wraps an existing error with additional context and details.
func WrapWithDetails(err error, code int, message, details string) *APIError {
	apiErr := &APIError{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: getCurrentTimestamp(),
		err:       err,
	}

	if err != nil {
		if apiErr.Details != "" {
			apiErr.Details = fmt.Sprintf("%s: %s", details, err.Error())
		} else {
			apiErr.Details = err.Error()
		}
	}

	return apiErr
}

// getCurrentTimestamp returns the current timestamp in RFC3339 format.
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// Is checks if the error is of type APIError with the given code.
func Is(err error, code int) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == code
	}
	return false
}

// As attempts to convert an error to *APIError.
func As(err error) (*APIError, bool) {
	apiErr, ok := err.(*APIError)
	return apiErr, ok
}

// IsAPIError checks if the error is an APIError using standard library errors.Is.
func IsAPIError(err error, target *APIError) bool {
	return stderrors.Is(err, target)
}

// AsAPIError attempts to convert an error to *APIError using standard library errors.As.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	ok := stderrors.As(err, &apiErr)
	return apiErr, ok
}

// UnwrapAPIError unwraps the error chain to find the original error.
func UnwrapAPIError(err error) error {
	return stderrors.Unwrap(err)
}

// GetCode returns the error code if the error is an APIError, otherwise returns 0.
func GetCode(err error) int {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code
	}
	return 0
}

// GetMessage returns the error message if the error is an APIError, otherwise returns the error string.
func GetMessage(err error) string {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Message
	}
	return err.Error()
}

// GetDetails returns the error details if the error is an APIError, otherwise returns empty string.
func GetDetails(err error) string {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Details
	}
	return ""
}

// String returns a string representation of the APIError.
func (e *APIError) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Code: %d", e.Code))
	parts = append(parts, fmt.Sprintf("Message: %s", e.Message))

	if e.Details != "" {
		parts = append(parts, fmt.Sprintf("Details: %s", e.Details))
	}

	if e.Timestamp != "" {
		parts = append(parts, fmt.Sprintf("Timestamp: %s", e.Timestamp))
	}

	return strings.Join(parts, "\n")
}

// WithDetails adds or updates the details of the error.
func (e *APIError) WithDetails(details string) *APIError {
	e.Details = details
	return e
}

// Wrap wraps another error with this APIError, creating a new APIError with the same code and message.
// The wrapped error becomes the underlying error and its message is appended to Details.
// This is useful for adding context to errors while maintaining the error chain.
func (e *APIError) Wrap(err error) *APIError {
	if err == nil {
		return e
	}

	newErr := &APIError{
		Code:      e.Code,
		Message:   e.Message,
		Timestamp: getCurrentTimestamp(),
		err:       err,
	}

	// Append the wrapped error's message to Details
	if e.Details != "" {
		newErr.Details = fmt.Sprintf("%s: %s", e.Details, err.Error())
	} else {
		newErr.Details = err.Error()
	}

	return newErr
}
