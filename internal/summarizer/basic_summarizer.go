package summarizer

import (
	"strings"
)

// BasicSummarizer is a simple implementation of the Summarizer interface.
// It extracts the first few sentences from the text as a summary.
type BasicSummarizer struct {
	maxSummaryLen int
}

// NewBasicSummarizer creates a new BasicSummarizer instance.
func NewBasicSummarizer(maxSummaryLen int) *BasicSummarizer {
	if maxSummaryLen <= 0 {
		maxSummaryLen = 200 // Default max summary length
	}
	return &BasicSummarizer{
		maxSummaryLen: maxSummaryLen,
	}
}

// Initialize sets up the summarizer with any required configuration.
func (s *BasicSummarizer) Initialize() error {
	return nil // No initialization needed for the basic summarizer
}

// Summarize takes a text input and returns a condensed summary.
// This basic implementation simply truncates the text to a specified length
// and attempts to end at a sentence boundary.
func (s *BasicSummarizer) Summarize(text string) (string, error) {
	if len(text) <= s.maxSummaryLen {
		return text, nil
	}

	// Calculate actual truncation length to leave room for ellipsis if needed
	ellipsis := "..."
	truncateLen := s.maxSummaryLen

	// Try to find a sentence boundary near the max length
	truncated := text[:truncateLen]

	// Look for common sentence terminators
	lastPeriod := strings.LastIndex(truncated, ".")
	lastQuestion := strings.LastIndex(truncated, "?")
	lastExclamation := strings.LastIndex(truncated, "!")

	// Find the last sentence boundary
	lastSentenceBoundary := max(lastPeriod, max(lastQuestion, lastExclamation))

	if lastSentenceBoundary > 0 {
		// End at the sentence boundary
		return text[:lastSentenceBoundary+1], nil
	}

	// If no sentence boundary found, find the last space
	// Adjust truncation length to leave room for ellipsis
	truncateLen = s.maxSummaryLen - len(ellipsis)
	if truncateLen < 0 {
		truncateLen = 0 // Edge case for very small maxSummaryLen
	}

	if truncateLen < len(text) {
		truncated = text[:truncateLen]
	}

	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		// End at a word boundary
		return text[:lastSpace] + ellipsis, nil
	}

	// If no good boundary found, just truncate and add ellipsis
	// Ensure that truncateLen + len(ellipsis) doesn't exceed maxSummaryLen
	return truncated + ellipsis, nil
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
