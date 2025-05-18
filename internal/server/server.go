// Package server provides the MCP server implementation for the ProjectMemory service.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

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
	mcpServer  server.Server
}

// NewContextToolServer creates a new MCPContextToolServer instance.
func NewContextToolServer(store contextstore.ContextStore, summarizer summarizer.Summarizer, embedder vector.Embedder) *MCPContextToolServer {
	return &MCPContextToolServer{
		store:      store,
		summarizer: summarizer,
		embedder:   embedder,
	}
}

// Initialize initializes the server with dependencies and configurations.
func (s *MCPContextToolServer) Initialize() error {
	slog.Info("Initializing MCP Context Tool Server")

	if s.store == nil || s.summarizer == nil || s.embedder == nil {
		return errortypes.ConfigError(errors.New("missing dependencies"), "server initialization failed")
	}

	// Create the MCP server
	srv := server.NewServer("projectmemory")

	// Register save_context tool
	srv = srv.Tool(tools.ToolSaveContext, "Save context to the persistent memory store",
		s.handleSaveContext)

	// Register retrieve_context tool
	srv = srv.Tool(tools.ToolRetrieveContext, "Retrieve relevant context based on a query",
		s.handleRetrieveContext)

	// Register delete_context tool
	srv = srv.Tool(tools.ToolDeleteContext, "Delete a specific context entry by ID",
		s.handleDeleteContext)

	// Register clear_all_context tool
	srv = srv.Tool(tools.ToolClearAllContext, "Clear all context entries from the store",
		s.handleClearAllContext)

	// Register replace_context tool
	srv = srv.Tool(tools.ToolReplaceContext, "Replace an existing context entry with new content",
		s.handleReplaceContext)

	s.mcpServer = srv
	slog.Info("MCP Context Tool Server initialized successfully", "tool_count", 5)
	return nil
}

// Start starts the MCP server on the specified transport.
func (s *MCPContextToolServer) Start() error {
	if s.mcpServer == nil {
		return errortypes.ConfigError(errors.New("server not initialized"), "cannot start server")
	}

	slog.Info("Starting MCP Context Tool Server")

	// Start the server using stdio transport
	stdioServer := s.mcpServer.AsStdio()
	return stdioServer.Run()
}

// Stop gracefully shuts down the MCP server.
func (s *MCPContextToolServer) Stop() error {
	slog.Info("Stopping MCP Context Tool Server")
	// The server will exit when stdin is closed
	return nil
}

// handleSaveContext handles the save_context MCP tool call.
func (s *MCPContextToolServer) handleSaveContext(ctx *server.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
	slog.Info("Processing save_context request", "text_length", len(req.ContextText))

	response := tools.SaveContextResponse{
		Status: "success",
	}

	// Generate summary
	slog.Debug("Generating summary for save_context")
	summary, err := s.summarizer.Summarize(req.ContextText)
	if err != nil {
		err = errortypes.APIError(err, "failed to summarize text").
			WithField("text_length", len(req.ContextText))
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Create embedding
	slog.Debug("Creating embedding for save_context")
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		err = errortypes.APIError(err, "failed to create embedding").
			WithField("summary_length", len(summary))
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		err = errortypes.APIError(err, "failed to convert embedding to bytes").
			WithField("embedding_size", len(embedding))
		errortypes.LogError(nil, err)

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
	slog.Debug("Storing context for save_context", "id", id)
	err = s.store.Store(id, summary, embeddingBytes, timestamp)
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to store context").
			WithField("context_id", id)
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Set response
	response.ID = id
	slog.Info("Successfully saved context", "id", id)

	// Return response
	return response, nil
}

// handleRetrieveContext handles the retrieve_context MCP tool call.
func (s *MCPContextToolServer) handleRetrieveContext(ctx *server.Context, req tools.RetrieveContextRequest) (tools.RetrieveContextResponse, error) {
	slog.Info("Processing retrieve_context request", "query", req.Query, "limit", req.Limit)

	response := tools.RetrieveContextResponse{
		Status: "success",
	}

	// Set default limit if not specified
	limit := req.Limit
	if limit <= 0 {
		limit = tools.DefaultRetrieveLimit
		slog.Debug("Using default limit for retrieve_context", "limit", limit)
	}

	// Create embedding for query
	slog.Debug("Creating embedding for query in retrieve_context")
	queryEmbedding, err := s.embedder.CreateEmbedding(req.Query)
	if err != nil {
		err = errortypes.APIError(err, "failed to create embedding for query").
			WithField("query", req.Query)
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Search context store
	slog.Debug("Searching context store for retrieve_context")
	results, err := s.store.Search(queryEmbedding, limit)
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to search context store").
			WithField("limit", limit)
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Set response
	response.Results = results
	slog.Info("Successfully retrieved context results", "count", len(results))

	// Return response
	return response, nil
}

// handleDeleteContext handles the delete_context MCP tool call.
func (s *MCPContextToolServer) handleDeleteContext(ctx *server.Context, req tools.DeleteContextRequest) (tools.DeleteContextResponse, error) {
	slog.Info("Processing delete_context request", "id", req.ID)

	response := tools.DeleteContextResponse{
		Status: "success",
	}

	// Delete context entry
	err := s.store.Delete(req.ID)
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to delete context").
			WithField("context_id", req.ID)
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	slog.Info("Successfully deleted context", "id", req.ID)

	// Return response
	return response, nil
}

// handleClearAllContext handles the clear_all_context MCP tool call.
func (s *MCPContextToolServer) handleClearAllContext(ctx *server.Context, req tools.ClearAllContextRequest) (tools.ClearAllContextResponse, error) {
	slog.Info("Processing clear_all_context request")

	response := tools.ClearAllContextResponse{
		Status: "success",
	}

	// Check confirmation string
	if req.Confirmation != "confirm" {
		response.Status = "error"
		response.Error = "Confirmation required. Set confirmation to 'confirm' to proceed with clearing all context"
		slog.Warn("Clear all context operation rejected: missing confirmation")
		return response, nil
	}

	// Clear all entries from context store
	count, err := s.store.Clear()
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to clear context store")
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	slog.Info("Successfully cleared context entries", "count", count)
	response.DeletedCount = count

	// Return response
	return response, nil
}

// handleReplaceContext handles the replace_context MCP tool call.
func (s *MCPContextToolServer) handleReplaceContext(ctx *server.Context, req tools.ReplaceContextRequest) (tools.ReplaceContextResponse, error) {
	slog.Info("Processing replace_context request", "id", req.ID, "new_text_length", len(req.ContextText))

	response := tools.ReplaceContextResponse{
		Status: "success",
	}

	// Validate ID
	if req.ID == "" {
		err := errortypes.ValidationError(errors.New("id cannot be empty for replace_context"), "invalid replace_context request")
		errortypes.LogError(nil, err)
		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Generate summary
	slog.Debug("Generating summary for replace_context")
	summary, err := s.summarizer.Summarize(req.ContextText)
	if err != nil {
		err = errortypes.APIError(err, "failed to summarize new text for replace_context").
			WithField("text_length", len(req.ContextText))
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Create embedding
	slog.Debug("Creating new embedding for replace_context")
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		err = errortypes.APIError(err, "failed to create new embedding for replace_context").
			WithField("summary_length", len(summary))
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		err = errortypes.APIError(err, "failed to convert new embedding to bytes for replace_context").
			WithField("embedding_size", len(embedding))
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	// Store (Replace) in context store
	slog.Debug("Replacing context for replace_context", "id", req.ID)
	timestamp := time.Now()
	err = s.store.Replace(req.ID, summary, embeddingBytes, timestamp)
	if err != nil {
		err = errortypes.DatabaseError(err, "failed to replace context for replace_context").
			WithField("context_id", req.ID)
		errortypes.LogError(nil, err)

		response.Status = "error"
		response.Error = err.Error()
		return response, nil
	}

	slog.Info("Successfully replaced context", "id", req.ID)

	// Return response
	return response, nil
}
