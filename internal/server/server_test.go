package server

import (
	"errors"
	"testing"
	"time"

	"github.com/localrivet/project-memory/internal/tools"
)

var testError = errors.New("test error")

// MockStore implements the contextstore.ContextStore interface for testing
type MockStore struct {
	StoredIDs        []string
	StoredSummaries  []string
	StoredEmbeddings [][]byte
	SearchResults    []string
	ReturnError      bool
}

func (m *MockStore) Initialize(dbPath string) error {
	if m.ReturnError {
		return testError
	}
	return nil
}

func (m *MockStore) Close() error {
	if m.ReturnError {
		return testError
	}
	return nil
}

func (m *MockStore) Store(id string, summaryText string, embedding []byte, timestamp time.Time) error {
	if m.ReturnError {
		return testError
	}
	m.StoredIDs = append(m.StoredIDs, id)
	m.StoredSummaries = append(m.StoredSummaries, summaryText)
	m.StoredEmbeddings = append(m.StoredEmbeddings, embedding)
	return nil
}

func (m *MockStore) Search(queryEmbedding []float32, limit int) ([]string, error) {
	if m.ReturnError {
		return nil, testError
	}

	if len(m.SearchResults) > limit {
		return m.SearchResults[:limit], nil
	}
	return m.SearchResults, nil
}

// MockSummarizer implements the summarizer.Summarizer interface for testing
type MockSummarizer struct {
	Summaries   map[string]string
	ReturnError bool
}

func (m *MockSummarizer) Initialize() error {
	if m.ReturnError {
		return testError
	}
	return nil
}

func (m *MockSummarizer) Summarize(text string) (string, error) {
	if m.ReturnError {
		return "", testError
	}

	if summary, exists := m.Summaries[text]; exists {
		return summary, nil
	}

	// Default behavior: return first 50 chars if not in map
	if len(text) > 50 {
		return text[:50] + "...", nil
	}
	return text, nil
}

// MockEmbedder implements the vector.Embedder interface for testing
type MockEmbedder struct {
	Embeddings  map[string][]float32
	ReturnError bool
}

func (m *MockEmbedder) Initialize() error {
	if m.ReturnError {
		return testError
	}
	return nil
}

func (m *MockEmbedder) CreateEmbedding(text string) ([]float32, error) {
	if m.ReturnError {
		return nil, testError
	}

	if embedding, exists := m.Embeddings[text]; exists {
		return embedding, nil
	}

	// Default behavior: return a simple embedding based on text length
	result := make([]float32, 4)
	for i := 0; i < 4 && i < len(text); i++ {
		result[i] = float32(text[i]) / 255.0
	}
	return result, nil
}

// TestSaveContext tests the save_context tool handler
func TestSaveContext(t *testing.T) {
	// Setup mocks
	mockStore := &MockStore{
		StoredIDs:        []string{},
		StoredSummaries:  []string{},
		StoredEmbeddings: [][]byte{},
	}

	mockSummarizer := &MockSummarizer{
		Summaries: map[string]string{
			"This is a test context": "Test context summary",
		},
	}

	mockEmbedder := &MockEmbedder{
		Embeddings: map[string][]float32{
			"Test context summary": {0.1, 0.2, 0.3, 0.4},
		},
	}

	// Create server
	server := NewContextToolServer(mockStore, mockSummarizer, mockEmbedder)
	err := server.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	// Create request
	req := tools.SaveContextRequest{
		ContextText: "This is a test context",
	}

	// Call handler directly
	response, err := server.handleSaveContext(nil, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
	if response.ID == "" {
		t.Error("Expected non-empty ID")
	}

	// Verify store was called
	if len(mockStore.StoredSummaries) != 1 {
		t.Fatalf("Expected 1 stored summary, got %d", len(mockStore.StoredSummaries))
	}
	if mockStore.StoredSummaries[0] != "Test context summary" {
		t.Errorf("Expected summary 'Test context summary', got '%s'", mockStore.StoredSummaries[0])
	}
}

// TestRetrieveContext tests the retrieve_context tool handler
func TestRetrieveContext(t *testing.T) {
	// Setup mocks
	mockStore := &MockStore{
		SearchResults: []string{"Summary 1", "Summary 2", "Summary 3"},
	}

	mockSummarizer := &MockSummarizer{}

	mockEmbedder := &MockEmbedder{
		Embeddings: map[string][]float32{
			"test query": {0.5, 0.6, 0.7, 0.8},
		},
	}

	// Create server
	server := NewContextToolServer(mockStore, mockSummarizer, mockEmbedder)
	err := server.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	// Create request with limit
	req := tools.RetrieveContextRequest{
		Query: "test query",
		Limit: 2,
	}

	// Call handler directly
	response, err := server.handleRetrieveContext(nil, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}
	if len(response.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(response.Results))
	}
	if response.Results[0] != "Summary 1" || response.Results[1] != "Summary 2" {
		t.Errorf("Results don't match expected values: %v", response.Results)
	}
}

// TestErrorHandling tests error handling in the tool handlers
func TestErrorHandling(t *testing.T) {
	// Test cases for different error scenarios
	testCases := []struct {
		name            string
		storeError      bool
		summarizerError bool
		embedderError   bool
		tool            string
	}{
		{"Store Error", true, false, false, "save"},
		{"Summarizer Error", false, true, false, "save"},
		{"Embedder Error", false, false, true, "save"},
		{"Store Error Retrieve", true, false, false, "retrieve"},
		{"Embedder Error Retrieve", false, false, true, "retrieve"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks with appropriate errors
			mockStore := &MockStore{
				ReturnError:   tc.storeError,
				SearchResults: []string{"Summary 1"},
			}

			mockSummarizer := &MockSummarizer{
				ReturnError: tc.summarizerError,
			}

			mockEmbedder := &MockEmbedder{
				ReturnError: tc.embedderError,
			}

			// Create server
			server := NewContextToolServer(mockStore, mockSummarizer, mockEmbedder)
			server.Initialize()

			if tc.tool == "save" {
				// Test save_context
				req := tools.SaveContextRequest{
					ContextText: "Error test context",
				}

				response, err := server.handleSaveContext(nil, req)

				// We expect no direct error from handler
				if err != nil {
					t.Fatalf("Handler should not return error: %v", err)
				}

				// Error should be in response
				if response.Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", response.Status)
				}
				if response.Error == "" {
					t.Error("Expected non-empty error message")
				}
			} else {
				// Test retrieve_context
				req := tools.RetrieveContextRequest{
					Query: "Error test query",
				}

				response, err := server.handleRetrieveContext(nil, req)

				// We expect no direct error from handler
				if err != nil {
					t.Fatalf("Handler should not return error: %v", err)
				}

				// Error should be in response
				if response.Status != "error" {
					t.Errorf("Expected status 'error', got '%s'", response.Status)
				}
				if response.Error == "" {
					t.Error("Expected non-empty error message")
				}
			}
		})
	}
}
