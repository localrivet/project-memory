// Package server provides the MCP server implementation for the ProjectMemory service.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/localrivet/gomcp"
	"github.com/localrivet/gomcp/server"
	"github.com/localrivet/projectmemory/internal/contextstore"
	"github.com/localrivet/projectmemory/internal/logger"
	"github.com/localrivet/projectmemory/internal/summarizer"
	"github.com/localrivet/projectmemory/internal/tools"
	"github.com/localrivet/projectmemory/internal/vector"
)

// Common server error types
var (
	ErrServerNotInitialized = errors.New("server not initialized")
	ErrMissingDependencies  = errors.New("one or more required dependencies are nil")
)

// MCPContextToolServer implements the ContextToolServer interface
// for handling MCP tool calls related to context storage and retrieval.
type MCPContextToolServer struct {
	store      contextstore.ContextStore
	summarizer summarizer.Summarizer
	embedder   vector.Embedder
	mcpServer  *server.Server
	log        *logger.Logger
}

// NewContextToolServer creates a new MCPContextToolServer instance.
func NewContextToolServer(store contextstore.ContextStore, summarizer summarizer.Summarizer, embedder vector.Embedder) *MCPContextToolServer {
	return &MCPContextToolServer{
		store:      store,
		summarizer: summarizer,
		embedder:   embedder,
		log:        logger.GetLogger("server"),
	}
}

// Initialize initializes the server with dependencies and configurations.
func (s *MCPContextToolServer) Initialize() error {
	s.log.Info("Initializing MCP Context Tool Server")

	if s.store == nil || s.summarizer == nil || s.embedder == nil {
		return logger.ConfigError(ErrMissingDependencies, "server initialization failed")
	}

	// Create the MCP server
	s.mcpServer = gomcp.NewServer("projectmemory").
		// Register save_context tool
		Tool(tools.ToolSaveContext, "Save context to the persistent memory store",
			s.handleSaveContext).
		// Register retrieve_context tool
		Tool(tools.ToolRetrieveContext, "Retrieve relevant context based on a query",
			s.handleRetrieveContext).
		// Register delete_context tool
		Tool(tools.ToolDeleteContext, "Delete a specific context entry by ID",
			s.handleDeleteContext).
		// Register clear_all_context tool
		Tool(tools.ToolClearAllContext, "Clear all context entries from the store",
			s.handleClearAllContext).
		// Register replace_context tool
		Tool(tools.ToolReplaceContext, "Replace an existing context entry with new content",
			s.handleReplaceContext)

	s.log.Info("MCP Context Tool Server initialized successfully with %d tools", 5)
	return nil
}

// Start starts the MCP server on the specified transport.
func (s *MCPContextToolServer) Start() error {
	if s.mcpServer == nil {
		return logger.ConfigError(ErrServerNotInitialized, "cannot start server")
	}

	s.log.Info("Starting MCP Context Tool Server")

	// Start the server using stdio transport
	s.mcpServer.AsStdio().Run()

	return nil
}

// Stop gracefully shuts down the MCP server.
func (s *MCPContextToolServer) Stop() error {
	s.log.Info("Stopping MCP Context Tool Server")
	// The server will exit when stdin is closed
	return nil
}

// handleSaveContext handles the save_context MCP tool call.
func (s *MCPContextToolServer) handleSaveContext(ctx *server.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
	reqLogger := s.log.WithContext("save_context")
	reqLogger.Debug("Processing save_context request (text length: %d)", len(req.ContextText))

	response := tools.SaveContextResponse{
		Status: "success",
	}

	// Generate summary
	reqLogger.Debug("Generating summary")
	summary, err := s.summarizer.Summarize(req.ContextText)
	if err != nil {
		err = logger.APIError(err, "failed to summarize text").
			WithField("text_length", len(req.ContextText))
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Create embedding
	reqLogger.Debug("Creating embedding")
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		err = logger.APIError(err, "failed to create embedding").
			WithField("summary_length", len(summary))
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		err = logger.APIError(err, "failed to convert embedding to bytes").
			WithField("embedding_size", len(embedding))
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Generate ID (simple hash of content + timestamp)
	timestamp := time.Now()
	hasher := sha256.New()
	hasher.Write([]byte(summary))
	hasher.Write([]byte(timestamp.String()))
	id := hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars of the hash

	// Store in context store
	reqLogger.Debug("Storing context with ID: %s", id)
	err = s.store.Store(id, summary, embeddingBytes, timestamp)
	if err != nil {
		err = logger.DatabaseError(err, "failed to store context").
			WithField("context_id", id)
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Set response
	response.ID = id
	reqLogger.Info("Successfully saved context with ID: %s", id)

	// Return response
	return response, nil
}

// handleRetrieveContext handles the retrieve_context MCP tool call.
func (s *MCPContextToolServer) handleRetrieveContext(ctx *server.Context, req tools.RetrieveContextRequest) (tools.RetrieveContextResponse, error) {
	reqLogger := s.log.WithContext("retrieve_context")
	reqLogger.Debug("Processing retrieve_context request (query: %s, limit: %d)", req.Query, req.Limit)

	response := tools.RetrieveContextResponse{
		Status: "success",
	}

	// Set default limit if not specified
	limit := req.Limit
	if limit <= 0 {
		limit = tools.DefaultRetrieveLimit
		reqLogger.Debug("Using default limit: %d", limit)
	}

	// Create embedding for query
	reqLogger.Debug("Creating embedding for query")
	queryEmbedding, err := s.embedder.CreateEmbedding(req.Query)
	if err != nil {
		err = logger.APIError(err, "failed to create embedding for query").
			WithField("query", req.Query)
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Search context store
	reqLogger.Debug("Searching context store")
	results, err := s.store.Search(queryEmbedding, limit)
	if err != nil {
		err = logger.DatabaseError(err, "failed to search context store").
			WithField("limit", limit)
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Set response
	response.Results = results
	reqLogger.Info("Successfully retrieved %d context results", len(results))

	// Return response
	return response, nil
}

// handleDeleteContext handles the delete_context MCP tool call.
func (s *MCPContextToolServer) handleDeleteContext(ctx *server.Context, req tools.DeleteContextRequest) (tools.DeleteContextResponse, error) {
	reqLogger := s.log.WithContext("delete_context")
	reqLogger.Debug("Processing delete_context request (ID: %s)", req.ID)

	response := tools.DeleteContextResponse{
		Status: "success",
	}

	// Validate the ID
	if req.ID == "" {
		err := logger.ValidationError(errors.New("ID is required"), "missing ID parameter")
		logger.LogError(err)

		response.Status = "error"
		response.Error = "ID is required"
		return response, nil
	}

	// Delete the context entry
	err := s.store.DeleteContext(req.ID)
	if err != nil {
		err = logger.DatabaseError(err, "failed to delete context entry").
			WithField("context_id", req.ID)
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	reqLogger.Info("Successfully deleted context with ID: %s", req.ID)
	return response, nil
}

// handleClearAllContext handles the clear_all_context MCP tool call.
func (s *MCPContextToolServer) handleClearAllContext(ctx *server.Context, req tools.ClearAllContextRequest) (tools.ClearAllContextResponse, error) {
	reqLogger := s.log.WithContext("clear_all_context")
	reqLogger.Debug("Processing clear_all_context request")

	response := tools.ClearAllContextResponse{
		Status: "success",
	}

	// Validate the confirmation
	if req.Confirmation != "confirm" {
		err := logger.ValidationError(
			errors.New("confirmation is required"),
			"confirmation must be 'confirm' to clear all context")
		logger.LogError(err)

		response.Status = "error"
		response.Error = "confirmation must be 'confirm' to clear all context"
		return response, nil
	}

	// Clear all context entries
	err := s.store.ClearAllContext()
	if err != nil {
		err = logger.DatabaseError(err, "failed to clear all context entries")
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	reqLogger.Info("Successfully cleared all context entries")
	return response, nil
}

// handleReplaceContext handles the replace_context MCP tool call.
func (s *MCPContextToolServer) handleReplaceContext(ctx *server.Context, req tools.ReplaceContextRequest) (tools.ReplaceContextResponse, error) {
	reqLogger := s.log.WithContext("replace_context")
	reqLogger.Debug("Processing replace_context request (ID: %s, text length: %d)", req.ID, len(req.ContextText))

	response := tools.ReplaceContextResponse{
		Status: "success",
	}

	// Validate inputs
	if req.ID == "" {
		err := logger.ValidationError(errors.New("ID is required"), "missing ID parameter")
		logger.LogError(err)

		response.Status = "error"
		response.Error = "ID is required"
		return response, nil
	}

	if req.ContextText == "" {
		err := logger.ValidationError(errors.New("context_text is required"), "missing context_text parameter")
		logger.LogError(err)

		response.Status = "error"
		response.Error = "context_text is required"
		return response, nil
	}

	// Generate summary
	reqLogger.Debug("Generating summary")
	summary, err := s.summarizer.Summarize(req.ContextText)
	if err != nil {
		err = logger.APIError(err, "failed to summarize text").
			WithField("text_length", len(req.ContextText))
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Create embedding
	reqLogger.Debug("Creating embedding")
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		err = logger.APIError(err, "failed to create embedding").
			WithField("summary_length", len(summary))
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		err = logger.APIError(err, "failed to convert embedding to bytes").
			WithField("embedding_size", len(embedding))
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Replace in context store
	reqLogger.Debug("Replacing context with ID: %s", req.ID)
	err = s.store.ReplaceContext(req.ID, summary, embeddingBytes, time.Now())
	if err != nil {
		err = logger.DatabaseError(err, "failed to replace context entry").
			WithField("context_id", req.ID)
		logger.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	reqLogger.Info("Successfully replaced context with ID: %s", req.ID)
	return response, nil
}
