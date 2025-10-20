package errors

import "net/http"

// Predefined API errors with common HTTP status codes
var (
	// 4xx Client Errors
	ErrBadRequest = &APIError{
		Code:    http.StatusBadRequest,
		Message: "Bad Request",
		Details: "The request could not be understood or was missing required parameters",
	}

	ErrUnauthorized = &APIError{
		Code:    http.StatusUnauthorized,
		Message: "Unauthorized",
		Details: "Authentication is required and has failed or has not been provided",
	}

	ErrForbidden = &APIError{
		Code:    http.StatusForbidden,
		Message: "Forbidden",
		Details: "The server understood the request but refuses to authorize it",
	}

	ErrNotFound = &APIError{
		Code:    http.StatusNotFound,
		Message: "Not Found",
		Details: "The requested resource could not be found",
	}

	ErrMethodNotAllowed = &APIError{
		Code:    http.StatusMethodNotAllowed,
		Message: "Method Not Allowed",
		Details: "The method specified in the request is not allowed for the resource",
	}

	ErrConflict = &APIError{
		Code:    http.StatusConflict,
		Message: "Conflict",
		Details: "The request could not be completed due to a conflict with the current state of the resource",
	}

	ErrUnprocessableEntity = &APIError{
		Code:    http.StatusUnprocessableEntity,
		Message: "Unprocessable Entity",
		Details: "The request was well-formed but was unable to be followed due to semantic errors",
	}

	ErrTooManyRequests = &APIError{
		Code:    http.StatusTooManyRequests,
		Message: "Too Many Requests",
		Details: "The user has sent too many requests in a given amount of time",
	}

	// 5xx Server Errors
	ErrInternalServer = &APIError{
		Code:    http.StatusInternalServerError,
		Message: "Internal Server Error",
		Details: "An unexpected error occurred on the server",
	}

	ErrNotImplemented = &APIError{
		Code:    http.StatusNotImplemented,
		Message: "Not Implemented",
		Details: "The server does not support the functionality required to fulfill the request",
	}

	ErrBadGateway = &APIError{
		Code:    http.StatusBadGateway,
		Message: "Bad Gateway",
		Details: "The server received an invalid response from an upstream server",
	}

	ErrServiceUnavailable = &APIError{
		Code:    http.StatusServiceUnavailable,
		Message: "Service Unavailable",
		Details: "The server is currently unable to handle the request due to temporary overloading or maintenance",
	}

	ErrGatewayTimeout = &APIError{
		Code:    http.StatusGatewayTimeout,
		Message: "Gateway Timeout",
		Details: "The server did not receive a timely response from an upstream server",
	}
)

// Business logic errors
var (
	ErrValidationFailed = &APIError{
		Code:    http.StatusBadRequest,
		Message: "Validation Failed",
		Details: "The provided data failed validation",
	}

	ErrResourceExists = &APIError{
		Code:    http.StatusConflict,
		Message: "Resource Already Exists",
		Details: "A resource with the same identifier already exists",
	}

	ErrResourceLocked = &APIError{
		Code:    http.StatusLocked,
		Message: "Resource Locked",
		Details: "The resource is locked and cannot be modified",
	}

	ErrInsufficientPermissions = &APIError{
		Code:    http.StatusForbidden,
		Message: "Insufficient Permissions",
		Details: "You do not have sufficient permissions to perform this action",
	}

	ErrRateLimitExceeded = &APIError{
		Code:    http.StatusTooManyRequests,
		Message: "Rate Limit Exceeded",
		Details: "You have exceeded the rate limit for this operation",
	}
)

// External service errors
var (
	ErrExternalServiceUnavailable = &APIError{
		Code:    http.StatusServiceUnavailable,
		Message: "External Service Unavailable",
		Details: "An external service is currently unavailable",
	}

	ErrExternalServiceTimeout = &APIError{
		Code:    http.StatusGatewayTimeout,
		Message: "External Service Timeout",
		Details: "The external service did not respond within the expected time",
	}

	ErrExternalServiceError = &APIError{
		Code:    http.StatusBadGateway,
		Message: "External Service Error",
		Details: "An error occurred while communicating with an external service",
	}
)

// NewBadRequest creates a new bad request error with custom details.
func NewBadRequest(details string) *APIError {
	return NewWithDetails(http.StatusBadRequest, "Bad Request", details)
}

// NewUnauthorized creates a new unauthorized error with custom details.
func NewUnauthorized(details string) *APIError {
	return NewWithDetails(http.StatusUnauthorized, "Unauthorized", details)
}

// NewForbidden creates a new forbidden error with custom details.
func NewForbidden(details string) *APIError {
	return NewWithDetails(http.StatusForbidden, "Forbidden", details)
}

// NewNotFound creates a new not found error with custom details.
func NewNotFound(details string) *APIError {
	return NewWithDetails(http.StatusNotFound, "Not Found", details)
}

// NewInternalServerError creates a new internal server error with custom details.
func NewInternalServerError(details string) *APIError {
	return NewWithDetails(http.StatusInternalServerError, "Internal Server Error", details)
}

// NewValidationError creates a new validation error with custom details.
func NewValidationError(details string) *APIError {
	return NewWithDetails(http.StatusBadRequest, "Validation Failed", details)
}

// NewExternalServiceError creates a new external service error with custom details.
func NewExternalServiceError(details string) *APIError {
	return NewWithDetails(http.StatusBadGateway, "External Service Error", details)
}
