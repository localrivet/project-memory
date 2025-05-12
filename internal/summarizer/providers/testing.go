package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockResponseConfig holds configuration for mock API responses
type MockResponseConfig struct {
	StatusCode   int
	ResponseBody interface{}
	Headers      map[string]string
}

// MockServer creates a test server that returns the configured response
func MockServer(t *testing.T, config MockResponseConfig) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers
		for k, v := range config.Headers {
			w.Header().Set(k, v)
		}

		// Always set content type if not explicitly set
		if _, exists := config.Headers["Content-Type"]; !exists {
			w.Header().Set("Content-Type", "application/json")
		}

		// Set status code
		w.WriteHeader(config.StatusCode)

		// Write response body
		if config.ResponseBody != nil {
			var respBytes []byte
			var err error

			// Handle string or []byte directly
			switch body := config.ResponseBody.(type) {
			case string:
				respBytes = []byte(body)
			case []byte:
				respBytes = body
			default:
				// Marshal other types to JSON
				respBytes, err = json.Marshal(body)
				if err != nil {
					t.Fatalf("Failed to marshal mock response: %v", err)
				}
			}

			if _, err := w.Write(respBytes); err != nil {
				t.Fatalf("Failed to write response body: %v", err)
			}
		}
	}))
}

// TestProvider is a simple implementation of LLMProvider for testing
type TestProvider struct {
	name         string
	returnError  error
	returnString string
}

// NewTestProvider creates a new TestProvider
func NewTestProvider(name string, returnString string, returnError error) *TestProvider {
	return &TestProvider{
		name:         name,
		returnString: returnString,
		returnError:  returnError,
	}
}

// Name returns the provider name
func (p *TestProvider) Name() string {
	return p.name
}

// Summarize returns the configured string or error
func (p *TestProvider) Summarize(_ context.Context, _ string, _ int) (string, error) {
	return p.returnString, p.returnError
}

// CapturingProvider is a provider that captures the inputs for testing
type CapturingProvider struct {
	name         string
	returnError  error
	returnString string
	capturedText string
	capturedMax  int
}

// NewCapturingProvider creates a new CapturingProvider
func NewCapturingProvider(name, returnString string, returnError error) *CapturingProvider {
	return &CapturingProvider{
		name:         name,
		returnString: returnString,
		returnError:  returnError,
	}
}

// Name returns the provider name
func (p *CapturingProvider) Name() string {
	return p.name
}

// Summarize captures inputs and returns configured response
func (p *CapturingProvider) Summarize(_ context.Context, text string, maxLength int) (string, error) {
	p.capturedText = text
	p.capturedMax = maxLength
	return p.returnString, p.returnError
}

// GetCapturedText returns the text that was passed to Summarize
func (p *CapturingProvider) GetCapturedText() string {
	return p.capturedText
}

// GetCapturedMaxLength returns the maxLength that was passed to Summarize
func (p *CapturingProvider) GetCapturedMaxLength() int {
	return p.capturedMax
}
