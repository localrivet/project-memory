package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/localrivet/project-memory/internal/logger"
)

func TestWriteErrorResponse(t *testing.T) {
	// Create a response recorder
	rr := httptest.NewRecorder()

	// Create an error and write the response
	err := errors.New("test error")
	writeErrorResponse(rr, http.StatusBadRequest, ErrorCodeInvalidRequest, "Invalid input", err)

	// Check the status code
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}

	// Check the content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type %s, got %s", "application/json", contentType)
	}

	// Decode the response
	var response ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	// Check the response fields
	if response.Status != http.StatusBadRequest {
		t.Errorf("Expected response status %d, got %d", http.StatusBadRequest, response.Status)
	}

	if response.Code != ErrorCodeInvalidRequest {
		t.Errorf("Expected response code %s, got %s", ErrorCodeInvalidRequest, response.Code)
	}

	if response.Message != "Invalid input" {
		t.Errorf("Expected message %s, got %s", "Invalid input", response.Message)
	}

	if response.Details != "test error" {
		t.Errorf("Expected details %s, got %s", "test error", response.Details)
	}
}

func TestHandleError(t *testing.T) {
	testCases := []struct {
		name          string
		err           error
		expectedCode  int
		expectedErrID string
	}{
		{
			name:          "Validation error",
			err:           logger.ValidationError(errors.New("invalid input"), "validation failed"),
			expectedCode:  http.StatusBadRequest,
			expectedErrID: ErrorCodeInvalidRequest,
		},
		{
			name:          "Permission error",
			err:           logger.PermissionError(errors.New("permission denied"), "unauthorized"),
			expectedCode:  http.StatusUnauthorized,
			expectedErrID: ErrorCodeAuthenticationError,
		},
		{
			name:          "Database error",
			err:           logger.DatabaseError(errors.New("db connection failed"), "database error"),
			expectedCode:  http.StatusInternalServerError,
			expectedErrID: ErrorCodeInternalError,
		},
		{
			name:          "Network error",
			err:           logger.NetworkError(errors.New("timeout"), "network error"),
			expectedCode:  http.StatusBadGateway,
			expectedErrID: ErrorCodeBadGateway,
		},
		{
			name:          "Custom status error",
			err:           NewErrorWithStatus(errors.New("not found"), http.StatusNotFound, ErrorCodeResourceNotFound, "Resource not found"),
			expectedCode:  http.StatusNotFound,
			expectedErrID: ErrorCodeResourceNotFound,
		},
		{
			name:          "Unspecified error",
			err:           errors.New("unknown error"),
			expectedCode:  http.StatusInternalServerError,
			expectedErrID: ErrorCodeInternalError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a response recorder
			rr := httptest.NewRecorder()

			// Handle the error
			HandleError(rr, tc.err)

			// Check the status code
			if rr.Code != tc.expectedCode {
				t.Errorf("Expected status code %d, got %d", tc.expectedCode, rr.Code)
			}

			// Decode the response
			var response ErrorResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Error decoding response: %v", err)
			}

			// Check the response error code
			if response.Code != tc.expectedErrID {
				t.Errorf("Expected error code %s, got %s", tc.expectedErrID, response.Code)
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
