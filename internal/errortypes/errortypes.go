// Package errortypes provides error types and handling for ProjectMemory.
package errortypes

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
)

// ErrorType represents the type of error that occurred
type ErrorType string

// Error types
const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeAPI        ErrorType = "api"
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeExternal   ErrorType = "external"
)

// AppError represents an application error with context
type AppError struct {
	Err       error
	Type      ErrorType
	Message   string
	StackInfo string
	Fields    map[string]interface{}
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Err.Error()
}

// Unwrap unwraps the error to support errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithField adds a field to the error for additional context
func (e *AppError) WithField(key string, value interface{}) *AppError {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// WithFields adds multiple fields to the error for additional context
func (e *AppError) WithFields(fields map[string]interface{}) *AppError {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

// captureStack captures the stack trace at the call site
func captureStack() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var builder strings.Builder
	for {
		frame, more := frames.Next()
		// Skip testing and standard library frames
		if !strings.Contains(frame.File, "testing/") && !strings.Contains(frame.File, "/go/src/") {
			fmt.Fprintf(&builder, "%s:%d %s\n", frame.File, frame.Line, frame.Function)
		}
		if !more {
			break
		}
	}
	return builder.String()
}

// newAppError creates a new AppError with the given type, underlying error, and message
func newAppError(errType ErrorType, err error, message string) *AppError {
	if err == nil {
		err = errors.New("unknown error")
	}

	return &AppError{
		Err:       err,
		Type:      errType,
		Message:   message,
		StackInfo: captureStack(),
		Fields:    make(map[string]interface{}),
	}
}

// ValidationError creates a new validation error
func ValidationError(err error, message string) *AppError {
	return newAppError(ErrorTypeValidation, err, message)
}

// PermissionError creates a new permission error
func PermissionError(err error, message string) *AppError {
	return newAppError(ErrorTypePermission, err, message)
}

// DatabaseError creates a new database error
func DatabaseError(err error, message string) *AppError {
	return newAppError(ErrorTypeDatabase, err, message)
}

// NetworkError creates a new network error
func NetworkError(err error, message string) *AppError {
	return newAppError(ErrorTypeNetwork, err, message)
}

// APIError creates a new API error
func APIError(err error, message string) *AppError {
	return newAppError(ErrorTypeAPI, err, message)
}

// ConfigError creates a new configuration error
func ConfigError(err error, message string) *AppError {
	return newAppError(ErrorTypeConfig, err, message)
}

// InternalError creates a new internal error
func InternalError(err error, message string) *AppError {
	return newAppError(ErrorTypeInternal, err, message)
}

// ExternalError creates a new external error
func ExternalError(err error, message string) *AppError {
	return newAppError(ErrorTypeExternal, err, message)
}

// LogError logs an AppError using the provided slog.Logger or the default slog logger.
// It logs the error message, type, stack trace, and any associated fields.
func LogError(logger *slog.Logger, err error) {
	if logger == nil {
		logger = slog.Default()
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		// Prepare arguments for structured logging
		args := []any{
			"type", string(appErr.Type),
			"original_error", appErr.Err.Error(),
		}
		if appErr.StackInfo != "" {
			args = append(args, "stack", appErr.StackInfo)
		}
		for k, v := range appErr.Fields {
			args = append(args, k, v)
		}
		logger.Error(appErr.Message, args...)
	} else {
		// For generic errors, log the error message and the error itself
		logger.Error(err.Error(), "error", err)
	}
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeValidation
	}
	return false
}

// IsPermissionError checks if an error is a permission error
func IsPermissionError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypePermission
	}
	return false
}

// IsDatabaseError checks if an error is a database error
func IsDatabaseError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeDatabase
	}
	return false
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeNetwork
	}
	return false
}
