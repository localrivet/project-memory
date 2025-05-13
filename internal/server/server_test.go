package server

import (
	"errors"
	"testing"
	"time"

	"github.com/localrivet/projectmemory/internal/tools"
)

var testError = errors.New("test error")

// MockStore implements the contextstore.ContextStore interface for testing
type MockStore struct {
	StoredIDs        []string
	StoredSummaries  []string
	StoredEmbeddings [][]byte
	SearchResults    []string
	DeletedIDs       []string
	ClearedAll       bool
	ClearedCount     int
	ReplacedIDs      []string
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

// DeleteContext implements the contextstore.ContextStore.DeleteContext method
func (m *MockStore) DeleteContext(id string) error {
	if m.ReturnError {
		return testError
	}
	m.DeletedIDs = append(m.DeletedIDs, id)
	return nil
}

// Delete implements the contextstore.ContextStore.Delete method
func (m *MockStore) Delete(id string) error {
	if m.ReturnError {
		return testError
	}
	m.DeletedIDs = append(m.DeletedIDs, id)
	return nil
}

// ClearAllContext implements the contextstore.ContextStore.ClearAllContext method (legacy)
func (m *MockStore) ClearAllContext() error {
	if m.ReturnError {
		return testError
	}
	m.ClearedAll = true
	return nil
}

// Clear implements the contextstore.ContextStore.Clear method
func (m *MockStore) Clear() (int, error) {
	if m.ReturnError {
		return 0, testError
	}
	m.ClearedAll = true
	return m.ClearedCount, nil
}

// ReplaceContext implements the contextstore.ContextStore.ReplaceContext method (legacy)
func (m *MockStore) ReplaceContext(id string, summaryText string, embedding []byte, timestamp time.Time) error {
	if m.ReturnError {
		return testError
	}
	m.ReplacedIDs = append(m.ReplacedIDs, id)
	// Since our mock implementation of Store just appends, we need to track replacements separately
	return m.Store(id, summaryText, embedding, timestamp)
}

// Replace implements the contextstore.ContextStore.Replace method
func (m *MockStore) Replace(id string, summaryText string, embedding []byte, timestamp time.Time) error {
	if m.ReturnError {
		return testError
	}
	m.ReplacedIDs = append(m.ReplacedIDs, id)
	// Since our mock implementation of Store just appends, we need to track replacements separately
	return m.Store(id, summaryText, embedding, timestamp)
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

// TestDeleteContext tests the delete_context tool handler
func TestDeleteContext(t *testing.T) {
	// Setup mocks
	mockStore := &MockStore{
		DeletedIDs: []string{},
	}
	mockSummarizer := &MockSummarizer{}
	mockEmbedder := &MockEmbedder{}

	// Create server
	server := NewContextToolServer(mockStore, mockSummarizer, mockEmbedder)
	err := server.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	// Create request
	req := tools.DeleteContextRequest{
		ID: "test-context-id",
	}

	// Call handler directly
	response, err := server.handleDeleteContext(nil, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Verify store was called with correct ID
	if len(mockStore.DeletedIDs) != 1 {
		t.Fatalf("Expected 1 deleted ID, got %d", len(mockStore.DeletedIDs))
	}
	if mockStore.DeletedIDs[0] != "test-context-id" {
		t.Errorf("Expected ID 'test-context-id', got '%s'", mockStore.DeletedIDs[0])
	}
}

// TestClearAllContext tests the clear_all_context tool handler
func TestClearAllContext(t *testing.T) {
	// Setup mocks
	mockStore := &MockStore{}
	mockSummarizer := &MockSummarizer{}
	mockEmbedder := &MockEmbedder{}

	// Create server
	server := NewContextToolServer(mockStore, mockSummarizer, mockEmbedder)
	err := server.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	// Create request
	req := tools.ClearAllContextRequest{
		Confirmation: "confirm", // Using the correct confirmation string
	}

	// Call handler directly
	response, err := server.handleClearAllContext(nil, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Verify store was called
	if !mockStore.ClearedAll {
		t.Fatalf("Expected ClearAllContext to be called on the store")
	}
}

// TestReplaceContext tests the replace_context tool handler
func TestReplaceContext(t *testing.T) {
	// Setup mocks
	mockStore := &MockStore{
		ReplacedIDs: []string{},
	}

	mockSummarizer := &MockSummarizer{
		Summaries: map[string]string{
			"This is updated context": "Updated context summary",
		},
	}

	mockEmbedder := &MockEmbedder{
		Embeddings: map[string][]float32{
			"Updated context summary": {0.5, 0.6, 0.7, 0.8},
		},
	}

	// Create server
	server := NewContextToolServer(mockStore, mockSummarizer, mockEmbedder)
	err := server.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	// Create request
	req := tools.ReplaceContextRequest{
		ID:          "existing-context-id",
		ContextText: "This is updated context",
	}

	// Call handler directly
	response, err := server.handleReplaceContext(nil, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", response.Status)
	}

	// Verify store.ReplaceContext was called with correct ID
	if len(mockStore.ReplacedIDs) != 1 {
		t.Fatalf("Expected 1 replaced ID, got %d", len(mockStore.ReplacedIDs))
	}
	if mockStore.ReplacedIDs[0] != "existing-context-id" {
		t.Errorf("Expected ID 'existing-context-id', got '%s'", mockStore.ReplacedIDs[0])
	}

	// Verify the content was summarized and embedded
	if len(mockStore.StoredSummaries) != 1 {
		t.Fatalf("Expected 1 stored summary, got %d", len(mockStore.StoredSummaries))
	}
	if mockStore.StoredSummaries[0] != "Updated context summary" {
		t.Errorf("Expected summary 'Updated context summary', got '%s'", mockStore.StoredSummaries[0])
	}
}

// TestClearAllContextWithoutConfirmation tests that clear_all_context requires confirmation
func TestClearAllContextWithoutConfirmation(t *testing.T) {
	// Setup mocks
	mockStore := &MockStore{}
	mockSummarizer := &MockSummarizer{}
	mockEmbedder := &MockEmbedder{}

	// Create server
	server := NewContextToolServer(mockStore, mockSummarizer, mockEmbedder)
	err := server.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	// Create request without confirmation
	req := tools.ClearAllContextRequest{
		Confirmation: "no", // Using string confirmation instead of boolean
	}

	// Call handler directly
	response, err := server.handleClearAllContext(nil, req)
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify response indicates error
	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}

	// Verify store was NOT called
	if mockStore.ClearedAll {
		t.Fatalf("ClearAllContext should not have been called without confirmation")
	}
}
