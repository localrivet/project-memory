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
	anthropicAPIURL = "https://api.anthropic.com/v1/messages"
)

// AnthropicProvider implements the LLMProvider interface for Anthropic's Claude
type AnthropicProvider struct {
	Config
	httpClient *http.Client
	version    string
}

// AnthropicMessage represents the request structure for Anthropic's API
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest represents a request to Anthropic's API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []AnthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
}

// AnthropicResponse represents a response from Anthropic's API
type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewAnthropicProvider creates a new instance of the Anthropic provider
func NewAnthropicProvider(config Config) *AnthropicProvider {
	return &AnthropicProvider{
		Config: config,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		version: "2023-06-01", // API version, can be made configurable
	}
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return ProviderAnthropic
}

// Summarize implements the LLMProvider interface for Anthropic
func (p *AnthropicProvider) Summarize(ctx context.Context, text string, maxLength int) (string, error) {
	if p.APIKey == "" {
		return "", fmt.Errorf("Anthropic API key not provided")
	}

	// Default to Claude 3 Haiku if no model specified
	model := p.ModelID
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	// Create the API request
	reqBody := AnthropicRequest{
		Model: model,
		Messages: []AnthropicMessage{
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
		anthropicAPIURL,
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.APIKey)
	req.Header.Set("Anthropic-Version", p.version)

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to Anthropic API: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Parse response
	var anthResponse AnthropicResponse
	if err := json.Unmarshal(respBody, &anthResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	// Check for API error
	if anthResponse.Error != nil {
		return "", fmt.Errorf("Anthropic API error: %s: %s",
			anthResponse.Error.Type, anthResponse.Error.Message)
	}

	// Extract summary
	if len(anthResponse.Content) == 0 || anthResponse.Content[0].Text == "" {
		return "", fmt.Errorf("empty response from Anthropic API")
	}

	summary := anthResponse.Content[0].Text
	if len(summary) > maxLength {
		summary = summary[:maxLength]
	}

	return summary, nil
}
