package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/localrivet/projectmemory/internal/errortypes"
)

// ErrorResponse represents the structure of error responses sent by the API
type ErrorResponse struct {
	Status     string                 `json:"status"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
}

// Common error codes
const (
	// ErrorCodeInvalidRequest indicates the client sent an invalid request
	ErrorCodeInvalidRequest = "INVALID_REQUEST"

	// ErrorCodeInternalError indicates an internal server error
	ErrorCodeInternalError = "INTERNAL_ERROR"

	// ErrorCodeAuthenticationError indicates an authentication failure
	ErrorCodeAuthenticationError = "AUTHENTICATION_ERROR"

	// ErrorCodeResourceNotFound indicates a requested resource was not found
	ErrorCodeResourceNotFound = "RESOURCE_NOT_FOUND"

	// ErrorCodeBadGateway indicates a failure in an upstream service
	ErrorCodeBadGateway = "BAD_GATEWAY"
)

// Error response codes
const (
	StatusCodeValidationError = "VALIDATION_ERROR"
	StatusCodePermissionError = "PERMISSION_ERROR"
	StatusCodeDatabaseError   = "DATABASE_ERROR"
	StatusCodeNetworkError    = "NETWORK_ERROR"
	StatusCodeInternalError   = "INTERNAL_ERROR"
	StatusCodeConfigError     = "CONFIG_ERROR"
	StatusCodeExternalError   = "EXTERNAL_ERROR"
	StatusCodeUnknownError    = "UNKNOWN_ERROR"
)

// writeErrorResponse writes a structured error response to the HTTP response writer
func writeErrorResponse(w http.ResponseWriter, status int, code, message string, err error) {
	// Create the error response
	errResp := ErrorResponse{
		Status:  "error",
		Code:    code,
		Message: message,
	}

	// Add details from the error if available
	if err != nil {
		errResp.Details = map[string]interface{}{
			"error": err.Error(),
		}

		// Log the error with structured context
		logErr := errortypes.APIError(err, fmt.Sprintf("API Error (%s)", code)).
			WithField("status_code", status).
			WithField("error_code", code).
			WithField("client_message", message)

		errortypes.LogError(nil, logErr)
	}

	// Set content type and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Encode and send the response
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		// If JSON encoding fails, fall back to plain text
		slog.Error("Failed to encode error response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// Error handler functions for common HTTP error scenarios

// HandleBadRequest handles 400 Bad Request errors
func HandleBadRequest(w http.ResponseWriter, message string, err error) {
	writeErrorResponse(w, http.StatusBadRequest, ErrorCodeInvalidRequest, message, err)
}

// HandleUnauthorized handles 401 Unauthorized errors
func HandleUnauthorized(w http.ResponseWriter, message string, err error) {
	writeErrorResponse(w, http.StatusUnauthorized, ErrorCodeAuthenticationError, message, err)
}

// HandleNotFound handles 404 Not Found errors
func HandleNotFound(w http.ResponseWriter, message string, err error) {
	writeErrorResponse(w, http.StatusNotFound, ErrorCodeResourceNotFound, message, err)
}

// HandleInternalError handles 500 Internal Server Error errors
func HandleInternalError(w http.ResponseWriter, message string, err error) {
	writeErrorResponse(w, http.StatusInternalServerError, ErrorCodeInternalError, message, err)
}

// HandleBadGateway handles 502 Bad Gateway errors
func HandleBadGateway(w http.ResponseWriter, message string, err error) {
	writeErrorResponse(w, http.StatusBadGateway, ErrorCodeBadGateway, message, err)
}

// ErrorWithStatus creates an error with an HTTP status code
type ErrorWithStatus struct {
	err        error
	statusCode int
	errorCode  string
	message    string
}

// NewErrorWithStatus creates a new error with HTTP status code
func NewErrorWithStatus(err error, status int, code, message string) *ErrorWithStatus {
	return &ErrorWithStatus{
		err:        err,
		statusCode: status,
		errorCode:  code,
		message:    message,
	}
}

// Error returns the error message
func (e *ErrorWithStatus) Error() string {
	if e.message != "" {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.err.Error()
}

// Unwrap returns the underlying error
func (e *ErrorWithStatus) Unwrap() error {
	return e.err
}

// StatusCode returns the HTTP status code
func (e *ErrorWithStatus) StatusCode() int {
	return e.statusCode
}

// ErrorCode returns the application error code
func (e *ErrorWithStatus) ErrorCode() string {
	return e.errorCode
}

// Message returns the client-friendly message
func (e *ErrorWithStatus) Message() string {
	return e.message
}

// HandleError handles any error, inspecting its type to determine the appropriate HTTP response
func HandleError(w http.ResponseWriter, err error) {
	// Check if it's our specialized error type
	var statusErr *ErrorWithStatus
	if se, ok := err.(*ErrorWithStatus); ok {
		statusErr = se
	}

	if statusErr != nil {
		// Use the status code and message from the error
		writeErrorResponse(w, statusErr.StatusCode(), statusErr.ErrorCode(),
			statusErr.Message(), statusErr.Unwrap())
		return
	}

	// Check for AppError
	var appErr *errortypes.AppError
	if errors.As(err, &appErr) {
		// Get the error type and handle accordingly
		errType := appErr.Type

		switch errType {
		case errortypes.ErrorTypeValidation:
			HandleBadRequest(w, "Invalid request parameters", err)
			return
		case errortypes.ErrorTypePermission:
			HandleUnauthorized(w, "Permission denied", err)
			return
		case errortypes.ErrorTypeNetwork:
			HandleBadGateway(w, "Network error", err)
			return
		case errortypes.ErrorTypeDatabase, errortypes.ErrorTypeInternal:
			HandleInternalError(w, "An unexpected error occurred", err)
			return
		case errortypes.ErrorTypeAPI, errortypes.ErrorTypeExternal:
			HandleBadGateway(w, "Downstream service error", err)
			return
		}
	}

	// Check for specific error types using helper functions
	if errortypes.IsValidationError(err) {
		HandleBadRequest(w, "Invalid request parameters", err)
		return
	}

	if errortypes.IsPermissionError(err) {
		HandleUnauthorized(w, "Permission denied", err)
		return
	}

	if errortypes.IsNetworkError(err) {
		HandleBadGateway(w, "Network error", err)
		return
	}

	if errortypes.IsDatabaseError(err) {
		HandleInternalError(w, "An unexpected error occurred", err)
		return
	}

	// Default to internal server error for unknown error types
	HandleInternalError(w, "An unexpected error occurred", err)
}

// WriteError writes an error response to the HTTP response writer
func WriteError(w http.ResponseWriter, err error, status int) {
	// Log the error
	slog.Error("API Error", "error", err, "status", status)

	// Check if it's a known error type
	errorResponse := errorToResponse(err)

	// Set the HTTP status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Write the error response as JSON
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		// If we can't encode the JSON, fall back to a simple text response
		slog.Error("Error encoding JSON error response", "error", err, "original_error_message", errorResponse.Message, "status", status)
		http.Error(w, errorResponse.Message, status)
	}
}

// errorToResponse converts an error to a standardized ErrorResponse
func errorToResponse(err error) ErrorResponse {
	var code string
	var details map[string]interface{}
	var stackTrace string
	message := err.Error()

	// Check if it's an AppError
	var appErr *errortypes.AppError
	if errors.As(err, &appErr) {
		// Set details from the app error
		details = appErr.Fields
		stackTrace = appErr.StackInfo

		// Set the error code based on the error type
		switch appErr.Type {
		case errortypes.ErrorTypeValidation:
			code = StatusCodeValidationError
		case errortypes.ErrorTypePermission:
			code = StatusCodePermissionError
		case errortypes.ErrorTypeNetwork:
			code = StatusCodeNetworkError
		case errortypes.ErrorTypeDatabase, errortypes.ErrorTypeInternal:
			code = StatusCodeInternalError
		case errortypes.ErrorTypeAPI, errortypes.ErrorTypeExternal:
			code = StatusCodeExternalError
		case errortypes.ErrorTypeConfig:
			code = StatusCodeConfigError
		default:
			code = StatusCodeUnknownError
		}
	} else {
		// Generic error, use unknown error code
		code = StatusCodeUnknownError
	}

	// Return the standardized error response
	return ErrorResponse{
		Status:     "error",
		Code:       code,
		Message:    message,
		Details:    details,
		StackTrace: stackTrace,
	}
}
