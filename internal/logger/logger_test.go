package logger

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a logger with custom configuration
	config := &Config{
		Level:       DEBUG,
		Format:      TEXT,
		Output:      &buf,
		DefaultTags: map[string]interface{}{"test": true},
	}
	logger := New(config)

	// Test different log levels
	logger.Debug("This is a debug message")
	if !strings.Contains(buf.String(), "DEBUG") || !strings.Contains(buf.String(), "This is a debug message") {
		t.Errorf("Expected debug message in log output, got: %s", buf.String())
	}

	buf.Reset()
	logger.Info("This is an info message")
	if !strings.Contains(buf.String(), "INFO") || !strings.Contains(buf.String(), "This is an info message") {
		t.Errorf("Expected info message in log output, got: %s", buf.String())
	}

	// Test with context
	buf.Reset()
	logger.WithContext("testContext").Warn("This is a warning")
	if !strings.Contains(buf.String(), "WARN") ||
		!strings.Contains(buf.String(), "This is a warning") ||
		!strings.Contains(buf.String(), "[testContext]") {
		t.Errorf("Expected warning with context in log output, got: %s", buf.String())
	}

	// Test with fields
	buf.Reset()
	logger.WithField("customField", "value").Error("This is an error")
	if !strings.Contains(buf.String(), "ERROR") ||
		!strings.Contains(buf.String(), "This is an error") ||
		!strings.Contains(buf.String(), "customField=value") {
		t.Errorf("Expected error with field in log output, got: %s", buf.String())
	}

	// Test JSON format
	buf.Reset()
	jsonLogger := New(&Config{
		Level:  INFO,
		Format: JSON,
		Output: &buf,
	})

	jsonLogger.Info("JSON message")
	if !strings.Contains(buf.String(), "\"level\":\"INFO\"") ||
		!strings.Contains(buf.String(), "\"message\":\"JSON message\"") {
		t.Errorf("Expected JSON formatted log, got: %s", buf.String())
	}
}

func TestErrorHandling(t *testing.T) {
	// Test basic error creation
	baseErr := errors.New("base error")
	appErr := ValidationError(baseErr, "validation failed")

	if appErr.Type != ErrorTypeValidation {
		t.Errorf("Expected error type %s, got %s", ErrorTypeValidation, appErr.Type)
	}

	if !strings.Contains(appErr.Error(), "validation failed") ||
		!strings.Contains(appErr.Error(), "base error") {
		t.Errorf("Error message incorrect: %s", appErr.Error())
	}

	// Test error wrapping
	wrappedErr := NetworkError(appErr, "network problem")
	if wrappedErr.Type != ErrorTypeNetwork {
		t.Errorf("Expected wrapped error type %s, got %s", ErrorTypeNetwork, wrappedErr.Type)
	}

	// Test error with fields
	fieldErr := APIError(baseErr, "API error").WithField("status", 500)
	if fieldErr.Fields["status"] != 500 {
		t.Errorf("Expected field value 500, got %v", fieldErr.Fields["status"])
	}

	// Test IsErrorType
	if !IsErrorType(fieldErr, ErrorTypeAPI) {
		t.Errorf("IsErrorType failed to identify correct error type")
	}

	if IsErrorType(fieldErr, ErrorTypeDatabase) {
		t.Errorf("IsErrorType incorrectly identified error type")
	}

	// Test error logging
	var buf bytes.Buffer
	testLogger := New(&Config{
		Level:  DEBUG,
		Format: TEXT,
		Output: &buf,
	})
	SetDefaultLogger(testLogger)

	LogError(fieldErr)
	if !strings.Contains(buf.String(), "API error") ||
		!strings.Contains(buf.String(), "error_type=api") {
		t.Errorf("Error not logged correctly: %s", buf.String())
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	// Create a logger with INFO level
	logger := New(&Config{
		Level:  INFO,
		Format: TEXT,
		Output: &buf,
	})

	// DEBUG should not be logged when level is INFO
	logger.Debug("Should not appear")
	if buf.Len() > 0 {
		t.Errorf("DEBUG message should not have been logged, got: %s", buf.String())
	}

	// INFO should be logged
	buf.Reset()
	logger.Info("Should appear")
	if buf.Len() == 0 {
		t.Errorf("INFO message should have been logged")
	}

	// Test level parsing
	if ParseLevel("DEBUG") != DEBUG {
		t.Errorf("Failed to parse DEBUG level")
	}

	if ParseLevel("unknown") != INFO {
		t.Errorf("Unknown level should default to INFO")
	}
}

func ExampleLogger_WithContext() {
	// This example shows how to use contextual logging
	var buf bytes.Buffer
	logger := New(&Config{
		Level:  DEBUG,
		Format: TEXT,
		Output: &buf,
	})

	// Create a component logger
	componentLogger := logger.WithContext("auth", "login")

	// Log with the component context
	componentLogger.Info("User login successful")

	// The output would include the context path [auth.login]
	fmt.Println("Contains context:", strings.Contains(buf.String(), "[auth.login]"))
	// Output: Contains context: true
}
