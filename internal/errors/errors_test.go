package errors

import (
	stderrors "errors"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiError *APIError
		expected string
	}{
		{
			name: "Error with details",
			apiError: &APIError{
				Code:    400,
				Message: "Bad Request",
				Details: "Invalid input",
			},
			expected: "[400] Bad Request: Invalid input",
		},
		{
			name: "Error without details",
			apiError: &APIError{
				Code:    404,
				Message: "Not Found",
			},
			expected: "[404] Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.apiError.Error(); got != tt.expected {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New(400, "Bad Request")

	if err.Code != 400 {
		t.Errorf("Expected code 400, got %d", err.Code)
	}

	if err.Message != "Bad Request" {
		t.Errorf("Expected message 'Bad Request', got '%s'", err.Message)
	}

	if err.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestNewWithDetails(t *testing.T) {
	err := NewWithDetails(400, "Bad Request", "Invalid input")

	if err.Code != 400 {
		t.Errorf("Expected code 400, got %d", err.Code)
	}

	if err.Message != "Bad Request" {
		t.Errorf("Expected message 'Bad Request', got '%s'", err.Message)
	}

	if err.Details != "Invalid input" {
		t.Errorf("Expected details 'Invalid input', got '%s'", err.Details)
	}
}

func TestWrap(t *testing.T) {
	originalErr := stderrors.New("original error")
	wrappedErr := Wrap(originalErr, 500, "Internal Server Error")

	if wrappedErr.Code != 500 {
		t.Errorf("Expected code 500, got %d", wrappedErr.Code)
	}

	if wrappedErr.Message != "Internal Server Error" {
		t.Errorf("Expected message 'Internal Server Error', got '%s'", wrappedErr.Message)
	}

	if wrappedErr.Details != "original error" {
		t.Errorf("Expected details 'original error', got '%s'", wrappedErr.Details)
	}
}

func TestWrapWithDetails(t *testing.T) {
	originalErr := stderrors.New("original error")
	wrappedErr := WrapWithDetails(originalErr, 500, "Internal Server Error", "Database connection failed")

	if wrappedErr.Code != 500 {
		t.Errorf("Expected code 500, got %d", wrappedErr.Code)
	}

	if wrappedErr.Message != "Internal Server Error" {
		t.Errorf("Expected message 'Internal Server Error', got '%s'", wrappedErr.Message)
	}

	expectedDetails := "Database connection failed: original error"
	if wrappedErr.Details != expectedDetails {
		t.Errorf("Expected details '%s', got '%s'", expectedDetails, wrappedErr.Details)
	}
}

func TestIs(t *testing.T) {
	err := New(404, "Not Found")

	if !Is(err, 404) {
		t.Error("Expected Is(err, 404) to return true")
	}

	if Is(err, 400) {
		t.Error("Expected Is(err, 400) to return false")
	}

	// Test with non-APIError
	regularErr := stderrors.New("regular error")
	if Is(regularErr, 404) {
		t.Error("Expected Is(regularErr, 404) to return false")
	}
}

func TestAs(t *testing.T) {
	err := New(404, "Not Found")

	apiErr, ok := As(err)
	if !ok {
		t.Error("Expected As(err) to return true")
	}

	if apiErr.Code != 404 {
		t.Errorf("Expected code 404, got %d", apiErr.Code)
	}

	// Test with non-APIError
	regularErr := stderrors.New("regular error")
	_, ok = As(regularErr)
	if ok {
		t.Error("Expected As(regularErr) to return false")
	}
}

func TestGetCode(t *testing.T) {
	err := New(404, "Not Found")

	if GetCode(err) != 404 {
		t.Errorf("Expected code 404, got %d", GetCode(err))
	}

	// Test with non-APIError
	regularErr := stderrors.New("regular error")
	if GetCode(regularErr) != 0 {
		t.Errorf("Expected code 0, got %d", GetCode(regularErr))
	}
}

func TestGetMessage(t *testing.T) {
	err := New(404, "Not Found")

	if GetMessage(err) != "Not Found" {
		t.Errorf("Expected message 'Not Found', got '%s'", GetMessage(err))
	}

	// Test with non-APIError
	regularErr := stderrors.New("regular error")
	if GetMessage(regularErr) != "regular error" {
		t.Errorf("Expected message 'regular error', got '%s'", GetMessage(regularErr))
	}
}

func TestGetDetails(t *testing.T) {
	err := NewWithDetails(400, "Bad Request", "Invalid input")

	if GetDetails(err) != "Invalid input" {
		t.Errorf("Expected details 'Invalid input', got '%s'", GetDetails(err))
	}

	// Test with non-APIError
	regularErr := stderrors.New("regular error")
	if GetDetails(regularErr) != "" {
		t.Errorf("Expected empty details, got '%s'", GetDetails(regularErr))
	}
}

func TestString(t *testing.T) {
	err := NewWithDetails(400, "Bad Request", "Invalid input")

	str := err.String()
	if str == "" {
		t.Error("Expected String() to return non-empty string")
	}

	// Check that all components are included
	if !contains(str, "Code: 400") {
		t.Error("Expected String() to contain code")
	}
	if !contains(str, "Message: Bad Request") {
		t.Error("Expected String() to contain message")
	}
	if !contains(str, "Details: Invalid input") {
		t.Error("Expected String() to contain details")
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected int
	}{
		{"BadRequest", ErrBadRequest, 400},
		{"Unauthorized", ErrUnauthorized, 401},
		{"Forbidden", ErrForbidden, 403},
		{"NotFound", ErrNotFound, 404},
		{"InternalServer", ErrInternalServer, 500},
		{"ValidationFailed", ErrValidationFailed, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.expected {
				t.Errorf("Expected code %d, got %d", tt.expected, tt.err.Code)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) *APIError
		expected int
	}{
		{"NewBadRequest", NewBadRequest, 400},
		{"NewUnauthorized", NewUnauthorized, 401},
		{"NewForbidden", NewForbidden, 403},
		{"NewNotFound", NewNotFound, 404},
		{"NewInternalServerError", NewInternalServerError, 500},
		{"NewValidationError", NewValidationError, 400},
		{"NewExternalServiceError", NewExternalServiceError, 502},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn("test details")
			if err.Code != tt.expected {
				t.Errorf("Expected code %d, got %d", tt.expected, err.Code)
			}
			if err.Details != "test details" {
				t.Errorf("Expected details 'test details', got '%s'", err.Details)
			}
		})
	}
}

func TestStandardLibraryCompatibility(t *testing.T) {
	// Test stderrors.Is compatibility
	err1 := New(404, "Not Found")
	err2 := New(404, "Not Found")
	err3 := New(500, "Internal Error")

	if !stderrors.Is(err1, err2) {
		t.Error("stderrors.Is should return true for same APIError")
	}

	if stderrors.Is(err1, err3) {
		t.Error("stderrors.Is should return false for different APIError")
	}

	// Test stderrors.As compatibility
	var apiErr *APIError
	if !stderrors.As(err1, &apiErr) {
		t.Error("stderrors.As should return true for APIError")
	}

	if apiErr.Code != 404 {
		t.Errorf("Expected code 404, got %d", apiErr.Code)
	}

	// Test stderrors.Unwrap compatibility
	// APIError doesn't wrap errors, so Unwrap should return nil
	err4 := New(500, "Internal Error")
	unwrapped := stderrors.Unwrap(err4)
	if unwrapped != nil {
		t.Error("stderrors.Unwrap should return nil for APIError")
	}
}

func TestIsAPIError(t *testing.T) {
	err := New(404, "Not Found")
	target := New(404, "Not Found")
	different := New(500, "Internal Error")

	if !IsAPIError(err, target) {
		t.Error("IsAPIError should return true for same APIError")
	}

	if IsAPIError(err, different) {
		t.Error("IsAPIError should return false for different APIError")
	}
}

func TestAsAPIError(t *testing.T) {
	err := New(404, "Not Found")

	apiErr, ok := AsAPIError(err)
	if !ok {
		t.Error("AsAPIError should return true for APIError")
	}

	if apiErr.Code != 404 {
		t.Errorf("Expected code 404, got %d", apiErr.Code)
	}

	// Test with non-APIError
	regularErr := stderrors.New("regular error")
	_, ok = AsAPIError(regularErr)
	if ok {
		t.Error("AsAPIError should return false for non-APIError")
	}
}

func TestUnwrapAPIError(t *testing.T) {
	originalErr := stderrors.New("original error")
	wrappedErr := Wrap(originalErr, 500, "Internal Error")

	// Now APIError wraps errors, so Unwrap should return the original error
	unwrapped := UnwrapAPIError(wrappedErr)
	if unwrapped != originalErr {
		t.Errorf("UnwrapAPIError should return original error, got %v", unwrapped)
	}

	// Test with simple error (no wrapped error)
	simpleErr := New(404, "Not Found")
	unwrapped = UnwrapAPIError(simpleErr)
	if unwrapped != nil {
		t.Error("UnwrapAPIError should return nil for non-wrapped error")
	}
}

func TestWithDetails(t *testing.T) {
	err := New(404, "Not Found")

	err.WithDetails("Resource not found in database")
	if err.Details != "Resource not found in database" {
		t.Errorf("Expected details to be set, got '%s'", err.Details)
	}

	// Test updating details
	err.WithDetails("Updated details")
	if err.Details != "Updated details" {
		t.Errorf("Expected details to be updated, got '%s'", err.Details)
	}
}

// TestAPIError_As tests the As method of APIError
func TestAPIError_As(t *testing.T) {
	err := New(500, "Internal Error")

	// Test successful As
	var target *APIError
	if !err.As(&target) {
		t.Error("Expected As to return true for **APIError")
	}
	if target != err {
		t.Error("Expected target to be set to err")
	}

	// Test failed As with wrong type
	var wrongTarget *error
	if err.As(&wrongTarget) {
		t.Error("Expected As to return false for wrong type")
	}
}

// TestAPIError_Is_EdgeCases tests edge cases of the Is method
func TestAPIError_Is_EdgeCases(t *testing.T) {
	err1 := New(500, "Internal Error")
	err2 := New(500, "Internal Error")
	err3 := New(400, "Bad Request")

	// Test with same code and message
	if !err1.Is(err2) {
		t.Error("Expected Is to return true for errors with same code and message")
	}

	// Test with different code
	if err1.Is(err3) {
		t.Error("Expected Is to return false for errors with different code")
	}

	// Test with non-APIError
	stdErr := stderrors.New("standard error")
	if err1.Is(stdErr) {
		t.Error("Expected Is to return false for non-APIError")
	}
}

// TestWrapWithDetails_EmptyDetails tests WrapWithDetails with empty details
func TestWrapWithDetails_EmptyDetails(t *testing.T) {
	originalErr := stderrors.New("original error")

	// Test with empty details string
	err := WrapWithDetails(originalErr, 500, "Internal Error", "")

	if err.Details != "original error" {
		t.Errorf("Expected details to be 'original error', got '%s'", err.Details)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && containsMiddle(s, substr)
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestErrorChaining tests the error wrapping and unwrapping functionality
func TestErrorChaining(t *testing.T) {
	// Create a chain of wrapped errors
	rootErr := stderrors.New("root cause")
	midErr := Wrap(rootErr, 500, "Database error")
	topErr := Wrap(midErr, 404, "User not found")

	// Test unwrapping
	if stderrors.Unwrap(topErr) != midErr {
		t.Error("First unwrap should return midErr")
	}

	if stderrors.Unwrap(midErr) != rootErr {
		t.Error("Second unwrap should return rootErr")
	}

	// Test errors.Is with root error
	if !stderrors.Is(topErr, rootErr) {
		t.Error("errors.Is should find rootErr in the chain")
	}

	// Test errors.Is with mid error
	if !stderrors.Is(topErr, midErr) {
		t.Error("errors.Is should find midErr in the chain")
	}
}

// TestAPIError_Is_WithCode tests the new Is behavior
func TestAPIError_Is_WithCode(t *testing.T) {
	// Create predefined error
	ErrNotFound := New(404, "Not Found")

	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "Same instance",
			err:      ErrNotFound,
			target:   ErrNotFound,
			expected: true,
		},
		{
			name:     "Same code different instance",
			err:      New(404, "User not found"),
			target:   ErrNotFound,
			expected: true,
		},
		{
			name:     "Different code",
			err:      New(500, "Internal Error"),
			target:   ErrNotFound,
			expected: false,
		},
		{
			name:     "Wrapped error with same code",
			err:      Wrap(stderrors.New("original"), 404, "Not found"),
			target:   ErrNotFound,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stderrors.Is(tt.err, tt.target)
			if result != tt.expected {
				t.Errorf("errors.Is() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestWrappedErrorPreservation tests that wrapped errors are preserved
func TestWrappedErrorPreservation(t *testing.T) {
	// Test with standard library error
	rootErr := stderrors.New("sql: no rows")
	apiErr := Wrap(rootErr, 404, "User not found")

	// Should be able to check if it's the root error
	if !stderrors.Is(apiErr, rootErr) {
		t.Error("Should be able to identify root error through error chain")
	}

	// Test Unwrap
	if stderrors.Unwrap(apiErr) != rootErr {
		t.Error("Unwrap should return the root error")
	}
}

// TestAPIError_Wrap tests the Wrap method on APIError instances
func TestAPIError_Wrap(t *testing.T) {
	// Define a predefined error
	ErrNotFound := New(404, "Not Found")

	t.Run("Wrap with standard error", func(t *testing.T) {
		originalErr := stderrors.New("record not in database")
		wrappedErr := ErrNotFound.Wrap(originalErr)

		// Check code and message are preserved
		if wrappedErr.Code != 404 {
			t.Errorf("Code = %d, want 404", wrappedErr.Code)
		}
		if wrappedErr.Message != "Not Found" {
			t.Errorf("Message = %s, want 'Not Found'", wrappedErr.Message)
		}

		// Check details contains original error
		if wrappedErr.Details != "record not in database" {
			t.Errorf("Details = %s, want 'record not in database'", wrappedErr.Details)
		}

		// Check error chain
		if stderrors.Unwrap(wrappedErr) != originalErr {
			t.Error("Unwrap should return original error")
		}

		// Check errors.Is works
		if !stderrors.Is(wrappedErr, originalErr) {
			t.Error("errors.Is should find original error in chain")
		}
	})

	t.Run("Wrap with nil error", func(t *testing.T) {
		wrappedErr := ErrNotFound.Wrap(nil)

		// Should return the same error
		if wrappedErr != ErrNotFound {
			t.Error("Wrapping nil should return the same error")
		}
	})

	t.Run("Wrap with existing details", func(t *testing.T) {
		ErrWithDetails := New(400, "Bad Request").WithDetails("validation failed")
		originalErr := stderrors.New("email is invalid")
		wrappedErr := ErrWithDetails.Wrap(originalErr)

		// Details should combine both
		expected := "validation failed: email is invalid"
		if wrappedErr.Details != expected {
			t.Errorf("Details = %s, want %s", wrappedErr.Details, expected)
		}
	})

	t.Run("Wrap preserves error type comparison", func(t *testing.T) {
		originalErr := stderrors.New("database error")
		wrappedErr := ErrNotFound.Wrap(originalErr)

		// Should still match the predefined error by code
		if !stderrors.Is(wrappedErr, ErrNotFound) {
			t.Error("Wrapped error should match predefined error by code")
		}
	})

	t.Run("Chain multiple wraps", func(t *testing.T) {
		rootErr := stderrors.New("root cause")
		midErr := New(500, "Internal Error").Wrap(rootErr)
		topErr := New(404, "Not Found").Wrap(midErr)

		// Check error chain
		if !stderrors.Is(topErr, rootErr) {
			t.Error("Should find root error in chain")
		}
		if !stderrors.Is(topErr, midErr) {
			t.Error("Should find mid error in chain")
		}
	})
}
