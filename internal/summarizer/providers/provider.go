// Package providers contains implementations of different LLM providers
// for text summarization.
package providers

import (
	"context"
	"time"
)

const (
	// Provider constants
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	ProviderGoogle    = "google"
	ProviderXAI       = "xai"

	// Default settings
	DefaultTimeout        = 30 * time.Second
	DefaultMaxInputLength = 8000
)

// LLMProvider defines the interface for different LLM service providers
type LLMProvider interface {
	// Summarize takes a text input and returns a condensed summary
	Summarize(ctx context.Context, text string, maxLength int) (string, error)

	// Name returns the provider name
	Name() string
}

// Config holds common configuration for LLM providers
type Config struct {
	APIKey  string
	ModelID string
}
