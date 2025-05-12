package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	xaiAPIURL = "https://api.groq.com/openai/v1/chat/completions"
)

// XAIProvider implements the LLMProvider interface for X.AI's Grok
type XAIProvider struct {
	Config
	httpClient *http.Client
}

// XAIMessage represents a message in X.AI's chat format (OpenAI compatible)
type XAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// XAIRequest represents a request to X.AI's API (OpenAI compatible)
type XAIRequest struct {
	Model     string       `json:"model"`
	Messages  []XAIMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens"`
}

// XAIResponse represents a response from X.AI's API (OpenAI compatible)
type XAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// NewXAIProvider creates a new instance of the X.AI provider
func NewXAIProvider(config Config) *XAIProvider {
	return &XAIProvider{
		Config: config,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// Name returns the provider name
func (p *XAIProvider) Name() string {
	return ProviderXAI
}

// Summarize implements the LLMProvider interface for X.AI
func (p *XAIProvider) Summarize(ctx context.Context, text string, maxLength int) (string, error) {
	if p.APIKey == "" {
		return "", fmt.Errorf("X.AI API key not provided")
	}

	// Default to Grok-1 if no model specified
	model := p.ModelID
	if model == "" {
		model = "grok-1"
	}

	// Create the API request (similar to OpenAI format)
	reqBody := XAIRequest{
		Model: model,
		Messages: []XAIMessage{
			{
				Role:    "system",
				Content: "You are a precise summarizer that creates concise summaries of text.",
			},
			{
				Role: "user",
				Content: fmt.Sprintf(
					"Summarize the following text in a concise way, keeping the most important points. "+
						"The summary should be no more than %d characters:\n\n%s",
					maxLength, text),
			},
		},
		MaxTokens: 1024, // Reasonable default, can be made configurable
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		xaiAPIURL,
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to X.AI API: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Parse response
	var xaiResponse XAIResponse
	if err := json.Unmarshal(respBody, &xaiResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	// Check for API error
	if xaiResponse.Error != nil {
		return "", fmt.Errorf("X.AI API error: %s: %s",
			xaiResponse.Error.Type, xaiResponse.Error.Message)
	}

	// Extract summary
	if len(xaiResponse.Choices) == 0 || xaiResponse.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response from X.AI API")
	}

	summary := xaiResponse.Choices[0].Message.Content
	if len(summary) > maxLength {
		summary = summary[:maxLength]
	}

	return summary, nil
}
