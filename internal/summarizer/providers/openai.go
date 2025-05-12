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
	openaiAPIURL = "https://api.openai.com/v1/chat/completions"
)

// OpenAIProvider implements the LLMProvider interface for OpenAI's models
type OpenAIProvider struct {
	Config
	httpClient *http.Client
}

// OpenAIMessage represents a message in OpenAI's chat format
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequest represents a request to OpenAI's API
type OpenAIRequest struct {
	Model     string          `json:"model"`
	Messages  []OpenAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

// OpenAIResponse represents a response from OpenAI's API
type OpenAIResponse struct {
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

// NewOpenAIProvider creates a new instance of the OpenAI provider
func NewOpenAIProvider(config Config) *OpenAIProvider {
	return &OpenAIProvider{
		Config: config,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return ProviderOpenAI
}

// Summarize implements the LLMProvider interface for OpenAI
func (p *OpenAIProvider) Summarize(ctx context.Context, text string, maxLength int) (string, error) {
	if p.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key not provided")
	}

	// Default to GPT-3.5-turbo if no model specified
	model := p.ModelID
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	// Create the API request
	reqBody := OpenAIRequest{
		Model: model,
		Messages: []OpenAIMessage{
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
		openaiAPIURL,
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
		return "", fmt.Errorf("error sending request to OpenAI API: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Parse response
	var openaiResponse OpenAIResponse
	if err := json.Unmarshal(respBody, &openaiResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	// Check for API error
	if openaiResponse.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s: %s",
			openaiResponse.Error.Type, openaiResponse.Error.Message)
	}

	// Extract summary
	if len(openaiResponse.Choices) == 0 || openaiResponse.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response from OpenAI API")
	}

	summary := openaiResponse.Choices[0].Message.Content
	if len(summary) > maxLength {
		summary = summary[:maxLength]
	}

	return summary, nil
}
