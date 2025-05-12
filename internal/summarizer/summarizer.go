// Package summarizer provides interfaces and implementations for
// summarizing text content within the Project-Memory service.
package summarizer

const (
	// DefaultMaxSummaryLength defines the default maximum length for summaries.
	DefaultMaxSummaryLength = 500

	// DefaultPreserveKeyTerms indicates whether key terms should be preserved in summaries.
	DefaultPreserveKeyTerms = true
)

// Summarizer defines the interface for summarizing text content.
type Summarizer interface {
	// Summarize takes a text input and returns a condensed summary.
	Summarize(text string) (string, error)

	// Initialize sets up the summarizer with any required configuration.
	Initialize() error
}
