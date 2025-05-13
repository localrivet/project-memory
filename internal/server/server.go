// Package server provides the MCP server implementation for the ProjectMemory service.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/localrivet/gomcp"
	"github.com/localrivet/gomcp/logx"
	"github.com/localrivet/gomcp/server"
	"github.com/localrivet/projectmemory/internal/contextstore"
	"github.com/localrivet/projectmemory/internal/errortypes"
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
	logger     logx.Logger
}

// NewContextToolServer creates a new MCPContextToolServer instance.
func NewContextToolServer(store contextstore.ContextStore, summarizer summarizer.Summarizer, embedder vector.Embedder) *MCPContextToolServer {
	return &MCPContextToolServer{
		store:      store,
		summarizer: summarizer,
		embedder:   embedder,
		logger:     logx.NewLogger("info"),
	}
}

// Initialize initializes the server with dependencies and configurations.
func (s *MCPContextToolServer) Initialize() error {
	s.logger.Info("Initializing MCP Context Tool Server")

	if s.store == nil || s.summarizer == nil || s.embedder == nil {
		return errortypes.ConfigError(errors.New("missing dependencies"), "server initialization failed")
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

	s.logger.Info("MCP Context Tool Server initialized successfully with %d tools", 5)
	return nil
}

// Start starts the MCP server on the specified transport.
func (s *MCPContextToolServer) Start() error {
	if s.mcpServer == nil {
		return errortypes.ConfigError(errors.New("server not initialized"), "cannot start server")
	}

	s.logger.Info("Starting MCP Context Tool Server")

	// Start the server using stdio transport
	s.mcpServer.AsStdio().Run()

	return nil
}

// Stop gracefully shuts down the MCP server.
func (s *MCPContextToolServer) Stop() error {
	s.logger.Info("Stopping MCP Context Tool Server")
	// The server will exit when stdin is closed
	return nil
}

// WithLogger sets a custom logger for the server and passes it to the underlying gomcp server.
func (s *MCPContextToolServer) WithLogger(customLogger logx.Logger) {
	s.logger = customLogger

	// If the gomcp server is already initialized, update its logger
	if s.mcpServer != nil {
		s.mcpServer.WithLogger(customLogger)
	}
}

// handleSaveContext handles the save_context MCP tool call.
func (s *MCPContextToolServer) handleSaveContext(ctx *server.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
	s.logger.Debug("Processing save_context request (text length: %d)", len(req.ContextText))

	response := tools.SaveContextResponse{
		Status: "success",
	}

	// Generate summary
	s.logger.Debug("Generating summary")
	summary, err := s.summarizer.Summarize(req.ContextText)
	if err != nil {
		err = errortypes.APIError(err, "failed to summarize text").
			WithField("text_length", len(req.ContextText))
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Create embedding
	s.logger.Debug("Creating embedding")
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		err = errortypes.APIError(err, "failed to create embedding").
			WithField("summary_length", len(summary))
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		err = errortypes.APIError(err, "failed to convert embedding to bytes").
			WithField("embedding_size", len(embedding))
		errortypes.LogError(err)

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
	s.logger.Debug("Storing context with ID: %s", id)
	err = s.store.Store(id, summary, embeddingBytes, timestamp)
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to store context").
			WithField("context_id", id)
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Set response
	response.ID = id
	s.logger.Info("Successfully saved context with ID: %s", id)

	// Return response
	return response, nil
}

// handleRetrieveContext handles the retrieve_context MCP tool call.
func (s *MCPContextToolServer) handleRetrieveContext(ctx *server.Context, req tools.RetrieveContextRequest) (tools.RetrieveContextResponse, error) {
	s.logger.Debug("Processing retrieve_context request (query: %s, limit: %d)", req.Query, req.Limit)

	response := tools.RetrieveContextResponse{
		Status: "success",
	}

	// Set default limit if not specified
	limit := req.Limit
	if limit <= 0 {
		limit = tools.DefaultRetrieveLimit
		s.logger.Debug("Using default limit: %d", limit)
	}

	// Create embedding for query
	s.logger.Debug("Creating embedding for query")
	queryEmbedding, err := s.embedder.CreateEmbedding(req.Query)
	if err != nil {
		err = errortypes.APIError(err, "failed to create embedding for query").
			WithField("query", req.Query)
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Search context store
	s.logger.Debug("Searching context store")
	results, err := s.store.Search(queryEmbedding, limit)
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to search context store").
			WithField("limit", limit)
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Set response
	response.Results = results
	s.logger.Info("Successfully retrieved %d context results", len(results))

	// Return response
	return response, nil
}

// handleDeleteContext handles the delete_context MCP tool call.
func (s *MCPContextToolServer) handleDeleteContext(ctx *server.Context, req tools.DeleteContextRequest) (tools.DeleteContextResponse, error) {
	s.logger.Debug("Processing delete_context request (ID: %s)", req.ID)

	response := tools.DeleteContextResponse{
		Status: "success",
	}

	// Delete from context store
	err := s.store.Delete(req.ID)
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to delete context").
			WithField("context_id", req.ID)
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	s.logger.Info("Successfully deleted context with ID: %s", req.ID)

	// Return response
	return response, nil
}

// handleClearAllContext handles the clear_all_context MCP tool call.
func (s *MCPContextToolServer) handleClearAllContext(ctx *server.Context, req tools.ClearAllContextRequest) (tools.ClearAllContextResponse, error) {
	s.logger.Debug("Processing clear_all_context request")

	response := tools.ClearAllContextResponse{
		Status: "success",
	}

	// Check confirmation string
	if req.Confirmation != "confirm" {
		response.Status = "error"
		response.Error = "Confirmation required. Set confirmation to 'confirm' to proceed with clearing all context"
		s.logger.Warn("Clear all context operation rejected: missing confirmation")
		return response, nil
	}

	// Clear all entries from context store
	count, err := s.store.Clear()
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to clear all context").
			WithField("clear_all", true)
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	response.DeletedCount = count
	s.logger.Info("Successfully cleared all context (%d entries removed)", count)

	// Return response
	return response, nil
}

// handleReplaceContext handles the replace_context MCP tool call.
func (s *MCPContextToolServer) handleReplaceContext(ctx *server.Context, req tools.ReplaceContextRequest) (tools.ReplaceContextResponse, error) {
	s.logger.Debug("Processing replace_context request (ID: %s, text length: %d)", req.ID, len(req.ContextText))

	response := tools.ReplaceContextResponse{
		Status: "success",
	}

	// Generate summary
	s.logger.Debug("Generating summary for replacement")
	summary, err := s.summarizer.Summarize(req.ContextText)
	if err != nil {
		err = errortypes.APIError(err, "failed to summarize replacement text").
			WithField("text_length", len(req.ContextText))
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Create embedding
	s.logger.Debug("Creating embedding for replacement")
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		err = errortypes.APIError(err, "failed to create embedding for replacement").
			WithField("summary_length", len(summary))
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		err = errortypes.APIError(err, "failed to convert replacement embedding to bytes").
			WithField("embedding_size", len(embedding))
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Replace in context store
	timestamp := time.Now()
	err = s.store.Replace(req.ID, summary, embeddingBytes, timestamp)
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to replace context").
			WithField("context_id", req.ID)
		errortypes.LogError(err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	s.logger.Info("Successfully replaced context with ID: %s", req.ID)

	// Return response
	return response, nil
}
