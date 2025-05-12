package summarizer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/localrivet/projectmemory/internal/summarizer/providers"
)

// MockLLMProvider implements the providers.LLMProvider interface for testing
type MockLLMProvider struct {
	returnError   bool
	failureCount  int
	currentTries  int
	returnSummary string
}

// Summarize implements the providers.LLMProvider interface for testing
func (m *MockLLMProvider) Summarize(ctx context.Context, text string, maxLength int) (string, error) {
	// Simulate context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Simulate failures with eventual recovery if failureCount is set
	if m.returnError || (m.failureCount > 0 && m.currentTries < m.failureCount) {
		m.currentTries++
		return "", errors.New("mock summarization error")
	}

	// Use a default summary if none provided
	summary := m.returnSummary
	if summary == "" {
		summary = "This is a mock summary of the text."
	}

	// For testing max length enforcement
	if len(summary) > maxLength {
		summary = summary[:maxLength]
	}

	return summary, nil
}

// Name returns the provider name
func (m *MockLLMProvider) Name() string {
	return "mock"
}

// TestNewAISummarizer tests the creation of a new AISummarizer with various configurations
func TestNewAISummarizer(t *testing.T) {
	// Test with nil config (should use defaults)
	s1 := NewAISummarizer(nil)
	if s1.maxSummaryLength != DefaultMaxSummaryLength {
		t.Errorf("Expected default maxSummaryLength, got %d", s1.maxSummaryLength)
	}

	// Test with custom config
	config := &AISummarizerConfig{
		MaxSummaryLength: 100,
		Timeout:          5 * time.Second,
		MaxRetries:       2,
		RetryDelay:       1 * time.Second,
		CacheCapacity:    500,
		CacheTTL:         1 * time.Hour,
	}
	s2 := NewAISummarizer(config)
	if s2.maxSummaryLength != 100 {
		t.Errorf("Expected maxSummaryLength of 100, got %d", s2.maxSummaryLength)
	}
	if s2.timeout != 5*time.Second {
		t.Errorf("Expected timeout of 5s, got %v", s2.timeout)
	}
}

// TestAISummarizerCache tests the caching functionality
func TestAISummarizerCache(t *testing.T) {
	// Create a mock provider that returns a specific summary
	mockProvider := &MockLLMProvider{
		returnSummary: "This is a cached summary.",
	}

	// Create the summarizer with the mock provider
	config := &AISummarizerConfig{
		MaxSummaryLength: 100,
		CacheCapacity:    10,
		CacheTTL:         1 * time.Hour,
	}
	summarizer := NewAISummarizer(config)
	summarizer.provider = mockProvider
	summarizer.providerInitialized = true

	// First call should use the provider
	text := "This is some text to summarize."
	summary1, err := summarizer.Summarize(text)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if summary1 != "This is a cached summary." {
		t.Errorf("Expected 'This is a cached summary.', got '%s'", summary1)
	}

	// Change the provider's response to verify cache is used
	mockProvider.returnSummary = "This is a different summary."

	// Second call with same text should use cache
	summary2, err := summarizer.Summarize(text)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if summary2 != "This is a cached summary." {
		t.Errorf("Expected cache hit 'This is a cached summary.', got '%s'", summary2)
	}

	// Call with different text should use provider again
	text2 := "This is different text to summarize."
	summary3, err := summarizer.Summarize(text2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if summary3 != "This is a different summary." {
		t.Errorf("Expected 'This is a different summary.', got '%s'", summary3)
	}
}

// TestAISummarizerRetries tests the retry functionality
func TestAISummarizerRetries(t *testing.T) {
	// Create a mock provider that fails a certain number of times then succeeds
	mockProvider := &MockLLMProvider{
		failureCount:  2, // Fail twice, succeed on 3rd try
		returnSummary: "Success after retries",
	}

	// Create the summarizer with the mock provider and retry settings
	config := &AISummarizerConfig{
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond, // Short delay for testing
	}
	summarizer := NewAISummarizer(config)
	summarizer.provider = mockProvider
	summarizer.providerInitialized = true

	// Should succeed after retries
	summary, err := summarizer.Summarize("Test text")
	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}
	if summary != "Success after retries" {
		t.Errorf("Expected 'Success after retries', got '%s'", summary)
	}

	// Test direct call to summarizeWithRetries to verify it returns error when provider fails
	t.Run("Direct summarizeWithRetries failure", func(t *testing.T) {
		// Create a new mock provider that always fails
		failingProvider := &MockLLMProvider{
			returnError: true,
		}

		// Create a summarizer with the failing provider
		failConfig := &AISummarizerConfig{
			MaxRetries: 1, // Only retry once
			RetryDelay: 10 * time.Millisecond,
		}
		failSummarizer := NewAISummarizer(failConfig)
		failSummarizer.provider = failingProvider
		failSummarizer.providerInitialized = true

		// Call summarizeWithRetries directly to bypass fallback mechanisms
		ctx, cancel := context.WithTimeout(context.Background(), failSummarizer.timeout)
		defer cancel()

		_, err := failSummarizer.summarizeWithRetries(ctx, "Test direct failure")
		if err == nil {
			t.Fatalf("Expected error from summarizeWithRetries, got success")
		}
	})
}

// TestAISummarizerFallback tests the fallback functionality
func TestAISummarizerFallback(t *testing.T) {
	// Primary provider always fails
	primaryProvider := &MockLLMProvider{
		returnError: true,
	}

	// Fallback provider succeeds
	fallbackProvider := &MockLLMProvider{
		returnSummary: "Fallback summary",
	}

	// Create the summarizer with providers
	config := &AISummarizerConfig{
		MaxRetries: 1,
		RetryDelay: 10 * time.Millisecond,
	}
	summarizer := NewAISummarizer(config)
	summarizer.provider = primaryProvider
	summarizer.fallbackProviders = []providers.LLMProvider{fallbackProvider}
	summarizer.providerInitialized = true

	// Should use fallback provider
	summary, err := summarizer.Summarize("Test text")
	if err != nil {
		t.Fatalf("Expected success with fallback, got error: %v", err)
	}
	if summary != "Fallback summary" {
		t.Errorf("Expected 'Fallback summary', got '%s'", summary)
	}

	// Test basic summarizer as final fallback when all others fail
	fallbackProvider.returnError = true

	// Create a different test summarizer to avoid issues with the cache
	summarizer2 := NewAISummarizer(config)
	summarizer2.provider = primaryProvider
	summarizer2.fallbackProviders = []providers.LLMProvider{fallbackProvider}
	summarizer2.providerInitialized = true

	// Should use the basic summarizer fallback
	veryShortText := "Test"
	summary, err = summarizer2.Summarize(veryShortText)
	if err != nil {
		t.Fatalf("Expected success with basic summarizer fallback, got error: %v", err)
	}

	// The basic summarizer should return the text itself if it's short
	if summary != veryShortText {
		t.Errorf("Expected '%s' from basic summarizer, got '%s'", veryShortText, summary)
	}
}
