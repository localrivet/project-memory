package main

import (
	"log"
	"os"
	"time"

	"github.com/localrivet/gomcp/server"
	"github.com/localrivet/projectmemory/internal/contextstore"
	pmserver "github.com/localrivet/projectmemory/internal/server" // Import with alias to avoid conflict
	"github.com/localrivet/projectmemory/internal/summarizer"
	"github.com/localrivet/projectmemory/internal/tools"
	"github.com/localrivet/projectmemory/internal/util"
	"github.com/localrivet/projectmemory/internal/vector"
)

func main() {
	// Set up logging - use standard log package
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting combined MCP server...")

	// ====================================================================
	// OPTION 1: DIRECT COMPONENT USAGE (Recommended for embedding)
	// ====================================================================

	// Initialize ProjectMemory components directly
	store := contextstore.NewSQLiteContextStore()
	if err := store.Initialize(".projectmemory.db"); err != nil {
		log.Printf("Failed to initialize context store: %v", err)
		os.Exit(1)
	}
	defer store.Close()

	// Initialize the summarizer
	summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	if err := summ.Initialize(); err != nil {
		log.Printf("Failed to initialize summarizer: %v", err)
		os.Exit(1)
	}

	// Initialize the embedder
	emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
	if err := emb.Initialize(); err != nil {
		log.Printf("Failed to initialize embedder: %v", err)
		os.Exit(1)
	}

	// Sample direct usage (outside of MCP)
	// Uncomment to test direct component usage
	/*
		// Save context example
		testText := "This is a test context to save."
		summary, _ := summ.Summarize(testText)
		embedding, _ := emb.CreateEmbedding(summary)
		embeddingBytes, _ := vector.Float32SliceToBytes(embedding)
		id := util.GenerateHash(summary, time.Now().UnixNano())
		store.Store(id, summary, embeddingBytes, time.Now())
		log.Printf("Stored context with ID: %s", id)

		// Retrieve context example
		queryText := "test context"
		queryEmbedding, _ := emb.CreateEmbedding(queryText)
		results, _ := store.Search(queryEmbedding, 5)
		log.Printf("Retrieved %d results", len(results))
	*/

	// Initialize the ProjectMemory context tool server - BUT DON'T START IT
	// We're directly creating the concrete implementation to avoid type assertion issues
	contextToolServer := pmserver.NewContextToolServer(store, summ, emb)
	// The server uses the standard log package internally

	if err := contextToolServer.Initialize(); err != nil {
		log.Printf("Failed to initialize context tool server: %v", err)
		os.Exit(1)
	}

	// ====================================================================
	// OPTION 2: HELPER FUNCTION FROM PROJECTMEMORY PACKAGE
	// ====================================================================

	// Alternative: Use the CreateComponents helper function
	/*
		config := projectmemory.DefaultConfig()
		config.Store.SQLitePath = ".projectmemory-alt.db"
		// Pass nil for logger to use slog.Default()
		altStore, altSumm, altEmb, err := projectmemory.CreateComponents(config, nil)
		if err != nil {
			log.Printf("Failed to create components: %v", err)
			os.Exit(1)
		}
		defer altStore.Close()
	*/

	// ====================================================================
	// OPTION 3: FULL SERVER INITIALIZATION
	// ====================================================================

	// Alternatively, use the high-level Server API (but don't start it)
	/*
		config := projectmemory.DefaultConfig()
		pmServer, err := projectmemory.NewServer(projectmemory.ServerOptions{
			Config: config, // Use the pre-configured config object
		})
		if err != nil {
			log.Printf("Failed to create ProjectMemory server: %v", err)
			os.Exit(1)
		}

		// You can use the high-level methods
		id, err := pmServer.SaveContext("This is a test context from the high-level API")
		if err != nil {
			log.Printf("Failed to save context: %v", err)
		} else {
			log.Printf("Saved context with ID: %s", id)

			// And retrieve it
			results, err := pmServer.RetrieveContext("test query", 5)
			if err != nil {
				log.Printf("Failed to retrieve context: %v", err)
			} else {
				for i, result := range results {
					log.Printf("Result %d: %s", i+1, result)
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
	mcpServer := server.NewServer("combined-mcp-server")
	// No need to set logger for the MCP server since it will use the standard log

	// Register your own tools
	mcpServer = mcpServer.Tool("your_custom_tool", "Description of your custom tool",
		func(ctx *server.Context, req YourCustomRequest) (YourCustomResponse, error) {
			// Your tool implementation
			return YourCustomResponse{}, nil
		})

	// We need a different approach since the handleSaveContext and handleRetrieveContext
	// methods are not publicly exposed. Instead, we'll use the components directly.

	// Save context tool implementation
	mcpServer = mcpServer.Tool(tools.ToolSaveContext, "Save context to the persistent memory store",
		func(ctx *server.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
			response := tools.SaveContextResponse{
				Status: "success",
			}

			// Generate summary using our summarizer
			log.Printf("Generating summary for text (length: %d)", len(req.ContextText))
			summary, err := summ.Summarize(req.ContextText)
			if err != nil {
				log.Printf("Failed to summarize text: %v", err)
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			// Create embedding
			log.Printf("Creating embedding for summary")
			embedding, err := emb.CreateEmbedding(summary)
			if err != nil {
				log.Printf("Failed to create embedding: %v", err)
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			// Convert embedding to bytes
			embeddingBytes, err := vector.Float32SliceToBytes(embedding)
			if err != nil {
				log.Printf("Failed to convert embedding to bytes: %v", err)
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			// Store in context store
			id := util.GenerateHash(summary, time.Now().UnixNano())
			log.Printf("Storing context with ID: %s", id)
			err = store.Store(id, summary, embeddingBytes, time.Now())
			if err != nil {
				log.Printf("Failed to store context: %v", err)
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			log.Printf("Successfully saved context with ID: %s", id)
			response.ID = id
			return response, nil
		})

	// Retrieve context tool implementation
	mcpServer = mcpServer.Tool(tools.ToolRetrieveContext, "Retrieve relevant context based on a query",
		func(ctx *server.Context, req tools.RetrieveContextRequest) (tools.RetrieveContextResponse, error) {
			response := tools.RetrieveContextResponse{
				Status: "success",
			}

			// Create embedding for query
			log.Printf("Creating embedding for query: %s", req.Query)
			queryEmbedding, err := emb.CreateEmbedding(req.Query)
			if err != nil {
				log.Printf("Failed to create embedding for query: %v", err)
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			// Set default limit if not specified
			limit := req.Limit
			if limit <= 0 {
				limit = tools.DefaultRetrieveLimit
				log.Printf("Using default limit: %d", limit)
			}

			// Search context store
			log.Printf("Searching context store with limit: %d", limit)
			results, err := store.Search(queryEmbedding, limit)
			if err != nil {
				log.Printf("Failed to search context store: %v", err)
				response.Status = "error"
				response.Error = err.Error()
				return response, nil
			}

			log.Printf("Retrieved %d context results", len(results))
			response.Results = results
			return response, nil
		})

	// Start your combined MCP server
	log.Printf("Starting combined MCP server with ProjectMemory tools...")
	stdioServer := mcpServer.AsStdio()
	stdioServer.Run()
}

// Example custom request/response types for your own tools
type YourCustomRequest struct {
	Param1 string `json:"param1"`
	Param2 int    `json:"param2"`
}

type YourCustomResponse struct {
	Result string `json:"result"`
}
