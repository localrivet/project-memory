package main

import (
	"github.com/localrivet/gomcp"
	gomcpserver "github.com/localrivet/gomcp/server"

	// Import the project-memory packages

	"time"

	projectmemory "github.com/localrivet/project-memory"
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

	// ====================================================================
	// OPTION 1: DIRECT COMPONENT USAGE (Recommended for embedding)
	// ====================================================================

	// Initialize Project-Memory components directly
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

	// Sample direct usage (outside of MCP)
	// Uncomment to test direct component usage
	/*
		// Save context example
		testText := "This is a test context to save."
		summary, _ := summ.Summarize(testText)
		embedding, _ := emb.CreateEmbedding(summary)
		embeddingBytes, _ := vector.Float32SliceToBytes(embedding)
		id := projectmemory.GenerateHash(summary, time.Now().UnixNano())
		store.Store(id, summary, embeddingBytes, time.Now())
		fmt.Printf("Stored context with ID: %s\n", id)

		// Retrieve context example
		queryText := "test context"
		queryEmbedding, _ := emb.CreateEmbedding(queryText)
		results, _ := store.Search(queryEmbedding, 5)
		fmt.Println("Retrieved results:", results)
	*/

	// Initialize the Project-Memory context tool server - BUT DON'T START IT
	// We're directly creating the concrete implementation to avoid type assertion issues
	contextToolServer := pmserver.NewContextToolServer(store, summ, emb)
	if err := contextToolServer.Initialize(); err != nil {
		log.Fatal("Failed to initialize context tool server: %v", err)
	}

	// ====================================================================
	// OPTION 2: HELPER FUNCTION FROM PROJECTMEMORY PACKAGE
	// ====================================================================

	// Alternative: Use the CreateComponents helper function
	/*
		config := projectmemory.DefaultConfig()
		config.Database.Path = ".projectmemory-alt.db"
		altStore, altSumm, altEmb, err := projectmemory.CreateComponents(config)
		if err != nil {
			log.Fatal("Failed to create components: %v", err)
		}
		defer altStore.Close()
	*/

	// ====================================================================
	// OPTION 3: FULL SERVER INITIALIZATION
	// ====================================================================

	// Alternatively, use the high-level Server API (but don't start it)
	/*
		config := projectmemory.DefaultConfig()
		pmServer, err := projectmemory.NewServer(config)
		if err != nil {
			log.Fatal("Failed to create Project-Memory server: %v", err)
		}

		// You can use the high-level methods
		id, err := pmServer.SaveContext("This is a test context from the high-level API")
		if err != nil {
			log.Error("Failed to save context: %v", err)
		} else {
			log.Info("Saved context with ID: %s", id)

			// And retrieve it
			results, err := pmServer.RetrieveContext("test context", 5)
			if err != nil {
				log.Error("Failed to retrieve context: %v", err)
			} else {
				for i, result := range results {
					log.Info("Result %d: %s", i+1, result)
				}
			}
		}

		// Or access the components directly if needed
		store := pmServer.GetStore()
		summ := pmServer.GetSummarizer()
		emb := pmServer.GetEmbedder()
	*/

	// ====================================================================
	// MCP SERVER INTEGRATION
	// ====================================================================

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
			id := projectmemory.GenerateHash(summary, time.Now().UnixNano())
			err = store.Store(id, summary, embeddingBytes, time.Now())
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
