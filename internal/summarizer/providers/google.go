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
	googleAPIURL = "https://generativelanguage.googleapis.com/v1beta/models"
)

// GoogleProvider implements the LLMProvider interface for Google's Gemini models
type GoogleProvider struct {
	Config
	httpClient *http.Client
}

// GoogleContent represents content in Google's Gemini API format
type GoogleContent struct {
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}

// GoogleRequest represents a request to Google's Gemini API
type GoogleRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
		Role string `json:"role,omitempty"`
	} `json:"contents"`
	GenerationConfig struct {
		MaxOutputTokens int `json:"maxOutputTokens"`
	} `json:"generationConfig"`
}

// GoogleResponse represents a response from Google's Gemini API
type GoogleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// NewGoogleProvider creates a new instance of the Google provider
func NewGoogleProvider(config Config) *GoogleProvider {
	return &GoogleProvider{
		Config: config,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// Name returns the provider name
func (p *GoogleProvider) Name() string {
	return ProviderGoogle
}

// Summarize implements the LLMProvider interface for Google
func (p *GoogleProvider) Summarize(ctx context.Context, text string, maxLength int) (string, error) {
	if p.APIKey == "" {
		return "", fmt.Errorf("Google API key not provided")
	}

	// Default to Gemini Pro if no model specified
	model := p.ModelID
	if model == "" {
		model = "gemini-pro"
	}

	// Create the API request
	reqBody := GoogleRequest{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role,omitempty"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{
						Text: fmt.Sprintf(
							"Summarize the following text in a concise way, keeping the most important points. "+
								"The summary should be no more than %d characters:\n\n%s",
							maxLength, text),
					},
				},
				Role: "user",
			},
		},
		GenerationConfig: struct {
			MaxOutputTokens int `json:"maxOutputTokens"`
		}{
			MaxOutputTokens: 1024, // Reasonable default, can be made configurable
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// Create HTTP request with API key in the URL
	apiURL := fmt.Sprintf("%s/%s:generateContent?key=%s", googleAPIURL, model, p.APIKey)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		apiURL,
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to Google API: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Parse response
	var googleResponse GoogleResponse
	if err := json.Unmarshal(respBody, &googleResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	// Check for API error
	if googleResponse.Error != nil {
		return "", fmt.Errorf("Google API error: %s: %s",
			googleResponse.Error.Status, googleResponse.Error.Message)
	}

	// Extract summary
	if len(googleResponse.Candidates) == 0 ||
		len(googleResponse.Candidates[0].Content.Parts) == 0 ||
		googleResponse.Candidates[0].Content.Parts[0].Text == "" {
		return "", fmt.Errorf("empty response from Google API")
	}

	summary := googleResponse.Candidates[0].Content.Parts[0].Text
	if len(summary) > maxLength {
		summary = summary[:maxLength]
	}

	return summary, nil
}
