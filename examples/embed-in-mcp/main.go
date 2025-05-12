package main

import (
	"github.com/localrivet/gomcp"
	gomcpserver "github.com/localrivet/gomcp/server"

	// Import the project-memory packages
	"time"

	"github.com/localrivet/project-memory/internal/contextstore"
	"github.com/localrivet/project-memory/internal/logger"
	pmserver "github.com/localrivet/project-memory/internal/server" // Import with alias to avoid conflict
	"github.com/localrivet/project-memory/internal/summarizer"
	"github.com/localrivet/project-memory/internal/tools"
	"github.com/localrivet/project-memory/internal/vector"
)

func main() {
	// Set up logging
	log := logger.GetDefaultLogger()
	log.Info("Starting combined MCP server...")

	// Initialize Project-Memory components
	store := contextstore.NewSQLiteContextStore()
	if err := store.Initialize(".projectmemory.db"); err != nil {
		log.Fatal("Failed to initialize context store: %v", err)
	}
	defer store.Close()

	// Initialize the summarizer
	summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	if err := summ.Initialize(); err != nil {
		log.Fatal("Failed to initialize summarizer: %v", err)
	}

	// Initialize the embedder
	emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
	if err := emb.Initialize(); err != nil {
		log.Fatal("Failed to initialize embedder: %v", err)
	}

	// Initialize the Project-Memory context tool server - BUT DON'T START IT
	// We're directly creating the concrete implementation to avoid type assertion issues
	contextToolServer := pmserver.NewContextToolServer(store, summ, emb)
	if err := contextToolServer.Initialize(); err != nil {
		log.Fatal("Failed to initialize context tool server: %v", err)
	}

	// ==== METHOD 1: Using the components directly ====

	// Create your own MCP server
	mcpServer := gomcp.NewServer("combined-mcp-server").
		// Register your own tools
		Tool("your_custom_tool", "Description of your custom tool",
			func(ctx *gomcpserver.Context, req YourCustomRequest) (YourCustomResponse, error) {
				// Your tool implementation
				return YourCustomResponse{}, nil
			})

	// We need a different approach since the handleSaveContext and handleRetrieveContext
	// methods are not publicly exposed. Instead, we'll use the components directly.

	// Save context tool implementation
	mcpServer = mcpServer.Tool(tools.ToolSaveContext, "Save context to the persistent memory store",
		func(ctx *gomcpserver.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
			response := tools.SaveContextResponse{
				Status: "success",
			}

			// Generate summary using our summarizer
			summary, err := summ.Summarize(req.ContextText)
			if err != nil {
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			// Create embedding
			embedding, err := emb.CreateEmbedding(summary)
			if err != nil {
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			// Convert embedding to bytes
			embeddingBytes, err := vector.Float32SliceToBytes(embedding)
			if err != nil {
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			// Store in context store
			id := generateUniqueID(summary)                                  // You'd need to implement this
			err = store.Store(id, summary, embeddingBytes, getCurrentTime()) // And this
			if err != nil {
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			response.ID = id
			return response, nil
		})

	// Retrieve context tool implementation
	mcpServer = mcpServer.Tool(tools.ToolRetrieveContext, "Retrieve relevant context based on a query",
		func(ctx *gomcpserver.Context, req tools.RetrieveContextRequest) (tools.RetrieveContextResponse, error) {
			response := tools.RetrieveContextResponse{
				Status: "success",
			}

			// Create embedding for query
			queryEmbedding, err := emb.CreateEmbedding(req.Query)
			if err != nil {
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			// Set default limit if not specified
			limit := req.Limit
			if limit <= 0 {
				limit = tools.DefaultRetrieveLimit
			}

			// Search context store
			results, err := store.Search(queryEmbedding, limit)
			if err != nil {
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			response.Results = results
			return response, nil
		})

	// Start your combined MCP server
	log.Info("Starting combined MCP server with Project-Memory tools...")
	mcpServer.AsStdio().Run()
}

// Example custom request/response types for your own tools
type YourCustomRequest struct {
	Param1 string `json:"param1"`
	Param2 int    `json:"param2"`
}

type YourCustomResponse struct {
	Result string `json:"result"`
}

// Helper functions that would need to be implemented
func generateUniqueID(summary string) string {
	// Implementation would generate a unique ID based on content
	return "example-id-12345"
}

func getCurrentTime() time.Time {
	return time.Now()
}
