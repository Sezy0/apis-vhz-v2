package apierror

import (
	"encoding/json"
	"net/http"
)

// Error represents a structured API error response.
type Error struct {
	StatusCode int          `json:"-"`
	Code       string       `json:"code"`
	Message    string       `json:"message"`
	Details    []FieldError `json:"details,omitempty"`
}

// FieldError represents a validation error for a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Message
}

// WithDetails adds field-level error details.
func (e *Error) WithDetails(details ...FieldError) *Error {
	e.Details = details
	return e
}

// ToJSON converts the error to JSON bytes.
func (e *Error) ToJSON() []byte {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    e.Code,
			"message": e.Message,
		},
	}

	if len(e.Details) > 0 {
		response["error"].(map[string]interface{})["details"] = e.Details
	}

	data, _ := json.Marshal(response)
	return data
}

// BadRequest creates a 400 Bad Request error.
func BadRequest(message string) *Error {
	return &Error{
		StatusCode: http.StatusBadRequest,
		Code:       "BAD_REQUEST",
		Message:    message,
	}
}

// ValidationError creates a 400 error with validation details.
func ValidationError(message string, details ...FieldError) *Error {
	return &Error{
		StatusCode: http.StatusBadRequest,
		Code:       "VALIDATION_ERROR",
		Message:    message,
		Details:    details,
	}
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(message string) *Error {
	if message == "" {
		message = "Authentication required"
	}
	return &Error{
		StatusCode: http.StatusUnauthorized,
		Code:       "UNAUTHORIZED",
		Message:    message,
	}
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(message string) *Error {
	if message == "" {
		message = "Access denied"
	}
	return &Error{
		StatusCode: http.StatusForbidden,
		Code:       "FORBIDDEN",
		Message:    message,
	}
}

// NotFound creates a 404 Not Found error.
func NotFound(message string) *Error {
	if message == "" {
		message = "Resource not found"
	}
	return &Error{
		StatusCode: http.StatusNotFound,
		Code:       "NOT_FOUND",
		Message:    message,
	}
}

// Conflict creates a 409 Conflict error.
func Conflict(message string) *Error {
	return &Error{
		StatusCode: http.StatusConflict,
		Code:       "CONFLICT",
		Message:    message,
	}
}

// InternalError creates a 500 Internal Server Error.
func InternalError(message string) *Error {
	if message == "" {
		message = "An unexpected error occurred"
	}
	return &Error{
		StatusCode: http.StatusInternalServerError,
		Code:       "INTERNAL_ERROR",
		Message:    message,
	}
}

// ServiceUnavailable creates a 503 Service Unavailable error.
func ServiceUnavailable(message string) *Error {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	return &Error{
		StatusCode: http.StatusServiceUnavailable,
		Code:       "SERVICE_UNAVAILABLE",
		Message:    message,
	}
}
