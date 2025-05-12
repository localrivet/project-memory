package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/localrivet/project-memory/internal/logger"
)

// ErrorResponse represents the structure of error responses sent by the API
type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
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

// writeErrorResponse writes a structured error response to the HTTP response writer
func writeErrorResponse(w http.ResponseWriter, status int, code, message string, err error) {
	// Create the error response
	errResp := ErrorResponse{
		Status:  status,
		Message: message,
		Code:    code,
	}

	// Add details from the error if available
	if err != nil {
		errResp.Details = err.Error()

		// Log the error with structured context
		logErr := logger.APIError(err, fmt.Sprintf("API Error (%s)", code)).
			WithField("status_code", status).
			WithField("error_code", code).
			WithField("client_message", message)

		logger.LogError(logErr)
	}

	// Set content type and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Encode and send the response
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		// If JSON encoding fails, fall back to plain text
		logger.Error("Failed to encode error response: %v", err)
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

	// Check for specific error types from our logger package
	if logger.IsErrorType(err, logger.ErrorTypeValidation) {
		HandleBadRequest(w, "Invalid request parameters", err)
		return
	}

	if logger.IsErrorType(err, logger.ErrorTypePermission) {
		HandleUnauthorized(w, "Permission denied", err)
		return
	}

	if logger.IsErrorType(err, logger.ErrorTypeDatabase) ||
		logger.IsErrorType(err, logger.ErrorTypeInternal) {
		HandleInternalError(w, "An unexpected error occurred", err)
		return
	}

	if logger.IsErrorType(err, logger.ErrorTypeAPI) ||
		logger.IsErrorType(err, logger.ErrorTypeNetwork) ||
		logger.IsErrorType(err, logger.ErrorTypeExternal) {
		HandleBadGateway(w, "Downstream service error", err)
		return
	}

	// Default to internal server error for unknown error types
	HandleInternalError(w, "An unexpected error occurred", err)
}
