package logger

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents different categories of errors in the system
type ErrorType string

// Error types used throughout the system
const (
	// ErrorTypeUnknown is used when the error type is not specified
	ErrorTypeUnknown ErrorType = "unknown"

	// ErrorTypeValidation indicates an input validation error
	ErrorTypeValidation ErrorType = "validation"

	// ErrorTypeDatabase indicates a database-related error
	ErrorTypeDatabase ErrorType = "database"

	// ErrorTypeNetwork indicates a network-related error
	ErrorTypeNetwork ErrorType = "network"

	// ErrorTypeAPI indicates an API-related error
	ErrorTypeAPI ErrorType = "api"

	// ErrorTypePermission indicates a permission-related error
	ErrorTypePermission ErrorType = "permission"

	// ErrorTypeConfiguration indicates a configuration-related error
	ErrorTypeConfiguration ErrorType = "configuration"

	// ErrorTypeExternal indicates an error from an external system
	ErrorTypeExternal ErrorType = "external"

	// ErrorTypeInternal indicates an internal system error
	ErrorTypeInternal ErrorType = "internal"
)

// AppError represents a structured error with context
type AppError struct {
	// Original error that we're wrapping
	Err error

	// Error type category
	Type ErrorType

	// Message provides additional context
	Message string

	// Fields contains additional structured data about the error
	Fields map[string]interface{}

	// Stack contains the call stack when the error was created
	Stack []string
}

// NewError creates a new structured error
func NewError(err error, errType ErrorType, message string) *AppError {
	if err == nil {
		err = errors.New("no error specified")
	}

	// If the input is already our error type, we can enrich it
	var existingErr *AppError
	if errors.As(err, &existingErr) {
		// Create a new error wrapping the existing one
		newErr := &AppError{
			Err:     existingErr.Err,
			Type:    errType,
			Message: message + ": " + existingErr.Message,
			Fields:  make(map[string]interface{}),
			Stack:   getCallStack(2), // Skip this function and the caller
		}

		// Copy the existing fields
		for k, v := range existingErr.Fields {
			newErr.Fields[k] = v
		}

		return newErr
	}

	// Create a new error from a standard error
	return &AppError{
		Err:     err,
		Type:    errType,
		Message: message,
		Fields:  make(map[string]interface{}),
		Stack:   getCallStack(2), // Skip this function and the caller
	}
}

// WithField adds a field to the error
func (e *AppError) WithField(key string, value interface{}) *AppError {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// WithFields adds multiple fields to the error
func (e *AppError) WithFields(fields map[string]interface{}) *AppError {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

// Error returns the error string
func (e *AppError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// LogError logs the error with appropriate context
func LogError(err error) {
	var structuredErr *AppError
	if errors.As(err, &structuredErr) {
		// Log with structured context
		fields := structuredErr.Fields
		if fields == nil {
			fields = make(map[string]interface{})
		}

		// Add error type and stack
		fields["error_type"] = string(structuredErr.Type)

		// Only add the stack if it's not too large
		if len(structuredErr.Stack) > 0 {
			fields["stack"] = strings.Join(structuredErr.Stack[:min(3, len(structuredErr.Stack))], " > ")
		}

		// Log with the structured context
		GetDefaultLogger().WithFields(fields).Error(structuredErr.Error())
		return
	}

	// Plain error without context
	Error("Unstructured error: %v", err)
}

// Helper function to get a call stack
func getCallStack(skip int) []string {
	stack := make([]string, 0, 10)

	// Collect up to 10 stack frames
	for i := skip; i < skip+10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// Get the function name
		fn := runtime.FuncForPC(pc)
		funcName := "unknown"
		if fn != nil {
			funcName = fn.Name()
			// Trim the package path
			if idx := strings.LastIndex(funcName, "/"); idx >= 0 {
				funcName = funcName[idx+1:]
			}
		}

		// Format the stack entry
		entry := fmt.Sprintf("%s:%d:%s", truncatePath(file), line, funcName)
		stack = append(stack, entry)
	}

	return stack
}

// Helper to truncate file paths
func truncatePath(path string) string {
	// Keep only the last two segments of the path
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper functions to create errors of specific types

// ValidationError creates a validation error
func ValidationError(err error, message string) *AppError {
	return NewError(err, ErrorTypeValidation, message)
}

// DatabaseError creates a database error
func DatabaseError(err error, message string) *AppError {
	return NewError(err, ErrorTypeDatabase, message)
}

// NetworkError creates a network error
func NetworkError(err error, message string) *AppError {
	return NewError(err, ErrorTypeNetwork, message)
}

// APIError creates an API error
func APIError(err error, message string) *AppError {
	return NewError(err, ErrorTypeAPI, message)
}

// PermissionError creates a permission error
func PermissionError(err error, message string) *AppError {
	return NewError(err, ErrorTypePermission, message)
}

// ConfigError creates a configuration error
func ConfigError(err error, message string) *AppError {
	return NewError(err, ErrorTypeConfiguration, message)
}

// ExternalError creates an error from an external system
func ExternalError(err error, message string) *AppError {
	return NewError(err, ErrorTypeExternal, message)
}

// InternalError creates an internal system error
func InternalError(err error, message string) *AppError {
	return NewError(err, ErrorTypeInternal, message)
}

// IsErrorType checks if an error is of a specific ErrorType
func IsErrorType(err error, errType ErrorType) bool {
	var structuredErr *AppError
	if errors.As(err, &structuredErr) {
		return structuredErr.Type == errType
	}
	return false
}
