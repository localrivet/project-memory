package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/localrivet/projectmemory/internal/errortypes"
)

func TestWriteErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		code       string
		message    string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "basic error",
			status:     http.StatusBadRequest,
			code:       "BAD_REQUEST",
			message:    "Invalid input",
			err:        errors.New("test error"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "nil error",
			status:     http.StatusInternalServerError,
			code:       "INTERNAL_ERROR",
			message:    "Something went wrong",
			err:        nil,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder
			w := httptest.NewRecorder()

			// Call the function
			writeErrorResponse(w, tt.status, tt.code, tt.message, tt.err)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("writeErrorResponse() status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Parse the response
			var resp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("Failed to parse response: %v", err)
				return
			}

			// Check the error code
			if resp.Code != tt.wantCode {
				t.Errorf("writeErrorResponse() code = %v, want %v", resp.Code, tt.wantCode)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	// Test cases for different error types
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "validation error",
			err:        errortypes.ValidationError(errors.New("invalid input"), "validation failed"),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "permission error",
			err:        errortypes.PermissionError(errors.New("permission denied"), "unauthorized"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "database error",
			err:        errortypes.DatabaseError(errors.New("db connection failed"), "database error"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "network error",
			err:        errortypes.NetworkError(errors.New("timeout"), "network error"),
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "unknown error",
			err:        errors.New("generic error"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "error with status",
			err:        NewErrorWithStatus(errors.New("not found"), http.StatusNotFound, "NOT_FOUND", "Resource not found"),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder
			w := httptest.NewRecorder()

			// Call HandleError
			HandleError(w, tt.err)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("HandleError() status = %v, want %v", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestErrorWithStatus(t *testing.T) {
	baseErr := errors.New("base error")
	statusErr := NewErrorWithStatus(baseErr, http.StatusBadRequest, "TEST_ERROR", "Test error message")

	// Check the error message
	expectedMsg := "Test error message: base error"
	if statusErr.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, statusErr.Error())
	}

	// Check the status code
	if statusErr.StatusCode() != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, statusErr.StatusCode())
	}

	// Check the error code
	if statusErr.ErrorCode() != "TEST_ERROR" {
		t.Errorf("Expected error code %s, got %s", "TEST_ERROR", statusErr.ErrorCode())
	}

	// Check the message
	if statusErr.Message() != "Test error message" {
		t.Errorf("Expected message %s, got %s", "Test error message", statusErr.Message())
	}

	// Check unwrapping
	if statusErr.Unwrap() != baseErr {
		t.Errorf("Unwrap should return the base error")
	}
}
