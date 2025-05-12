package summarizer

import (
	"strings"
	"testing"
)

func TestBasicSummarizer_Initialize(t *testing.T) {
	summarizer := NewBasicSummarizer(100)
	err := summarizer.Initialize()
	if err != nil {
		t.Errorf("Initialize() error = %v, want nil", err)
	}
}

func TestNewBasicSummarizer(t *testing.T) {
	tests := []struct {
		name          string
		maxSummaryLen int
		want          int
	}{
		{
			name:          "positive value",
			maxSummaryLen: 150,
			want:          150,
		},
		{
			name:          "zero value",
			maxSummaryLen: 0,
			want:          200, // default value
		},
		{
			name:          "negative value",
			maxSummaryLen: -50,
			want:          200, // default value
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NewBasicSummarizer(test.maxSummaryLen)
			if got.maxSummaryLen != test.want {
				t.Errorf("NewBasicSummarizer(%v) = %v, want %v", test.maxSummaryLen, got.maxSummaryLen, test.want)
			}
		})
	}
}

func TestBasicSummarizer_Summarize(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		maxSummaryLen int
		want          string
		wantContains  string // for cases where exact match isn't needed
	}{
		{
			name:          "short text",
			text:          "This is a short text.",
			maxSummaryLen: 100,
			want:          "This is a short text.",
		},
		{
			name:          "text with sentence boundary",
			text:          "This is the first sentence. This is the second sentence that should be truncated.",
			maxSummaryLen: 30,
			want:          "This is the first sentence.",
		},
		{
			name:          "text with question mark boundary",
			text:          "Is this the first sentence? This is the second sentence that should be truncated.",
			maxSummaryLen: 30,
			want:          "Is this the first sentence?",
		},
		{
			name:          "text with exclamation mark boundary",
			text:          "This is the first sentence! This is the second sentence that should be truncated.",
			maxSummaryLen: 30,
			want:          "This is the first sentence!",
		},
		{
			name:          "text without sentence boundary but with space",
			text:          "This is a long text without any sentence boundary that should be truncated at a word boundary",
			maxSummaryLen: 30,
			wantContains:  "...",
		},
		{
			name:          "text without spaces",
			text:          "ThisIsALongTextWithoutAnySpacesOrSentenceBoundaries",
			maxSummaryLen: 10,
			wantContains:  "...",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			summarizer := NewBasicSummarizer(test.maxSummaryLen)
			got, err := summarizer.Summarize(test.text)

			if err != nil {
				t.Errorf("Summarize() error = %v, want nil", err)
				return
			}

			if test.want != "" && got != test.want {
				t.Errorf("Summarize() = %v, want %v", got, test.want)
			}

			if test.wantContains != "" && !strings.Contains(got, test.wantContains) {
				t.Errorf("Summarize() = %v, want to contain %v", got, test.wantContains)
			}

			// Check that the result is not longer than maxSummaryLen
			if len(got) > test.maxSummaryLen {
				t.Errorf("Summarize() result length = %v, want <= %v", len(got), test.maxSummaryLen)
			}
		})
	}
}
