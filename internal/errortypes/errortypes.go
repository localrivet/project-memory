// Package errortypes provides error types and utilities for the projectmemory service.
package errortypes

import (
	"errors"
	"fmt"
	"os"
)

// ErrorType represents the category of an error
type ErrorType string

// Error type constants
const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeAPI        ErrorType = "api"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeExternal   ErrorType = "external"
	ErrorTypeConfig     ErrorType = "config"
)

// TypedError is an error with an associated error type
type TypedError struct {
	err      error
	errType  ErrorType
	message  string
	metadata map[string]interface{}
}

// Error returns the error message
func (e *TypedError) Error() string {
	if e.message != "" {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.err.Error()
}

// Unwrap returns the underlying error
func (e *TypedError) Unwrap() error {
	return e.err
}

// Type returns the error type
func (e *TypedError) Type() ErrorType {
	return e.errType
}

// WithField adds metadata to the error
func (e *TypedError) WithField(key string, value interface{}) *TypedError {
	if e.metadata == nil {
		e.metadata = make(map[string]interface{})
	}
	e.metadata[key] = value
	return e
}

// GetField retrieves metadata from the error
func (e *TypedError) GetField(key string) (interface{}, bool) {
	if e.metadata == nil {
		return nil, false
	}
	val, ok := e.metadata[key]
	return val, ok
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errType ErrorType) bool {
	var typedErr *TypedError
	if errors.As(err, &typedErr) {
		return typedErr.errType == errType
	}
	return false
}

// ValidationError creates a validation error
func ValidationError(err error, message string) *TypedError {
	return &TypedError{
		err:     err,
		errType: ErrorTypeValidation,
		message: message,
	}
}

// PermissionError creates a permission error
func PermissionError(err error, message string) *TypedError {
	return &TypedError{
		err:     err,
		errType: ErrorTypePermission,
		message: message,
	}
}

// DatabaseError creates a database error
func DatabaseError(err error, message string) *TypedError {
	return &TypedError{
		err:     err,
		errType: ErrorTypeDatabase,
		message: message,
	}
}

// InternalError creates an internal error
func InternalError(err error, message string) *TypedError {
	return &TypedError{
		err:     err,
		errType: ErrorTypeInternal,
		message: message,
	}
}

// APIError creates an API error
func APIError(err error, message string) *TypedError {
	return &TypedError{
		err:     err,
		errType: ErrorTypeAPI,
		message: message,
	}
}

// NetworkError creates a network error
func NetworkError(err error, message string) *TypedError {
	return &TypedError{
		err:     err,
		errType: ErrorTypeNetwork,
		message: message,
	}
}

// ExternalError creates an external service error
func ExternalError(err error, message string) *TypedError {
	return &TypedError{
		err:     err,
		errType: ErrorTypeExternal,
		message: message,
	}
}

// ConfigError creates a configuration error
func ConfigError(err error, message string) *TypedError {
	return &TypedError{
		err:     err,
		errType: ErrorTypeConfig,
		message: message,
	}
}

// LogError logs an error to stderr, useful for quick error logging without a logger
func LogError(err error) {
	fmt.Fprintf(standardErrorWriter, "ERROR: %v\n", err)
}

// standardErrorWriter is used for the LogError function
var standardErrorWriter = &fmtErrorWriter{}

type fmtErrorWriter struct{}

func (w *fmtErrorWriter) Write(p []byte) (n int, err error) {
	return fmt.Fprintf(DefaultErrorOutput, "%s", p)
}

// DefaultErrorOutput is the default output for error logging
var DefaultErrorOutput = errorWriter(2)

// errorWriter creates a writer that writes to the Stderr
type errorWriter int

func (errorWriter) Write(p []byte) (n int, err error) {
	return fmt.Fprintf(os.Stderr, "%s", p)
}
