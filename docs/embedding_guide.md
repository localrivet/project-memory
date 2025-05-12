# Embedding Project-Memory in Your Application

This comprehensive guide explains how to embed Project-Memory in your Go applications, whether you're building a CLI tool, a web service, or integrating with an existing MCP server.

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [Integration Patterns](#integration-patterns)
- [Common Use Cases](#common-use-cases)
- [Advanced Topics](#advanced-topics)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/localrivet/project-memory"
    "github.com/localrivet/project-memory/internal/contextstore"
    "github.com/localrivet/project-memory/internal/summarizer"
    "github.com/localrivet/project-memory/internal/vector"
)

func main() {
    // Initialize components
    store := contextstore.NewSQLiteContextStore()
    store.Initialize(".memory.db")
    defer store.Close()

    summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
    summ.Initialize()

    emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
    emb.Initialize()

    // Store a context
    text := "Important information to remember: The API key needs to be rotated monthly."
    summary, _ := summ.Summarize(text)
    embedding, _ := emb.CreateEmbedding(summary)
    embeddingBytes, _ := vector.Float32SliceToBytes(embedding)
    id := projectmemory.GenerateHash(summary, time.Now().UnixNano())
    store.Store(id, summary, embeddingBytes, time.Now())
    fmt.Printf("Stored context with ID: %s\n", id)

    // Retrieve context
    query := "API key rotation"
    queryEmbedding, _ := emb.CreateEmbedding(query)
    results, _ := store.Search(queryEmbedding, 5)
    fmt.Println("Results:", results)
}
```

## Installation

### Prerequisites

- Go 1.20+
- SQLite

### Adding the Dependency

```bash
go get github.com/localrivet/project-memory
```

### Verify Installation

Create a simple test program to verify that Project-Memory can be imported:

```go
package main

import (
    "fmt"
    "github.com/localrivet/project-memory"
)

func main() {
    config := projectmemory.DefaultConfig()
    fmt.Printf("Default database path: %s\n", config.Database.Path)
}
```

Run it:

```bash
go run test.go
```

If it runs without errors, the installation is successful.

## Configuration

Project-Memory uses a JSON configuration file, but when used as an embedded library, you can configure it programmatically.

### Using Default Configuration

```go
config := projectmemory.DefaultConfig()
```

### Customizing Configuration

```go
config := projectmemory.DefaultConfig()
config.Database.Path = "./custom-memory.db"
config.Models.Provider = "anthropic"  // If you have an API key for a real provider
config.Models.ModelID = "claude-3-opus-20240229"
config.Models.MaxTokens = 2048
config.Models.Temperature = 0.3
config.Logging.Level = "DEBUG"
```

### Loading from File

If you prefer to use a configuration file:

```go
config, err := projectmemory.LoadConfig(".projectmemoryconfig")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

## Integration Patterns

Project-Memory offers three primary integration patterns:

### 1. Direct Component Usage (Recommended)

This approach provides the most flexibility and control. You directly instantiate and manage the core components.

```go
// Initialize components
store := contextstore.NewSQLiteContextStore()
store.Initialize(".memory.db")
defer store.Close()

summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
summ.Initialize()

emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
emb.Initialize()

// Use the components directly
// ...
```

### 2. Using CreateComponents Helper

This pattern simplifies initialization by using a helper function to create all components at once.

```go
config := projectmemory.DefaultConfig()
config.Database.Path = ".memory.db"

store, summ, emb, err := projectmemory.CreateComponents(config)
if err != nil {
    log.Fatalf("Failed to create components: %v", err)
}
defer store.Close()

// Use the components
// ...
```

### 3. High-Level Server API (Without Starting the Server)

This pattern uses the Server API for simplified operations but without starting the MCP server.

```go
config := projectmemory.DefaultConfig()
pmServer, err := projectmemory.NewServer(config)
if err != nil {
    log.Fatalf("Failed to create server: %v", err)
}

// Use high-level methods
id, err := pmServer.SaveContext("Important information to remember")
if err != nil {
    log.Printf("Error saving context: %v", err)
}

results, err := pmServer.RetrieveContext("information", 5)
if err != nil {
    log.Printf("Error retrieving context: %v", err)
}

// Access components if needed
store := pmServer.GetStore()
summ := pmServer.GetSummarizer()
emb := pmServer.GetEmbedder()

// Do NOT call pmServer.Start()
```

## Common Use Cases

### CLI Tool with Memory Capabilities

```go
package main

import (
    "fmt"
    "os"
    "time"

    "github.com/spf13/cobra"
    "github.com/localrivet/project-memory"
    "github.com/localrivet/project-memory/internal/contextstore"
    "github.com/localrivet/project-memory/internal/summarizer"
    "github.com/localrivet/project-memory/internal/vector"
)

var store contextstore.ContextStore
var summ summarizer.Summarizer
var emb vector.Embedder

func initComponents() {
    store = contextstore.NewSQLiteContextStore()
    store.Initialize(".memory.db")

    summ = summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
    summ.Initialize()

    emb = vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
    emb.Initialize()
}

func main() {
    // Initialize components
    initComponents()
    defer store.Close()

    var rootCmd = &cobra.Command{
        Use:   "mycli",
        Short: "A CLI with memory capabilities",
    }

    var saveCmd = &cobra.Command{
        Use:   "save [text]",
        Short: "Save text to memory",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            text := args[0]
            summary, err := summ.Summarize(text)
            if err != nil {
                fmt.Printf("Error summarizing text: %v\n", err)
                return
            }

            embedding, err := emb.CreateEmbedding(summary)
            if err != nil {
                fmt.Printf("Error creating embedding: %v\n", err)
                return
            }

            embeddingBytes, err := vector.Float32SliceToBytes(embedding)
            if err != nil {
                fmt.Printf("Error converting embedding: %v\n", err)
                return
            }

            id := projectmemory.GenerateHash(summary, time.Now().UnixNano())
            err = store.Store(id, summary, embeddingBytes, time.Now())
            if err != nil {
                fmt.Printf("Error storing context: %v\n", err)
                return
            }

            fmt.Printf("Stored context with ID: %s\n", id)
        },
    }

    var retrieveCmd = &cobra.Command{
        Use:   "retrieve [query]",
        Short: "Retrieve context from memory",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            query := args[0]

            queryEmbedding, err := emb.CreateEmbedding(query)
            if err != nil {
                fmt.Printf("Error creating query embedding: %v\n", err)
                return
            }

            limit, _ := cmd.Flags().GetInt("limit")
            results, err := store.Search(queryEmbedding, limit)
            if err != nil {
                fmt.Printf("Error searching context: %v\n", err)
                return
            }

            fmt.Println("Results:")
            for i, res := range results {
                fmt.Printf("%d: %s\n", i+1, res)
            }
        },
    }
    retrieveCmd.Flags().IntP("limit", "l", 5, "Maximum number of results to return")

    rootCmd.AddCommand(saveCmd, retrieveCmd)

    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

### Web Service with Memory Capabilities

```go
package main

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    "github.com/localrivet/project-memory"
    "github.com/localrivet/project-memory/internal/contextstore"
    "github.com/localrivet/project-memory/internal/summarizer"
    "github.com/localrivet/project-memory/internal/vector"
)

var store contextstore.ContextStore
var summ summarizer.Summarizer
var emb vector.Embedder

func initComponents() error {
    store = contextstore.NewSQLiteContextStore()
    if err := store.Initialize(".memory.db"); err != nil {
        return err
    }

    summ = summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
    if err := summ.Initialize(); err != nil {
        return err
    }

    emb = vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
    if err := emb.Initialize(); err != nil {
        return err
    }

    return nil
}

type SaveRequest struct {
    Text string `json:"text"`
}

type SaveResponse struct {
    ID     string `json:"id,omitempty"`
    Status string `json:"status"`
    Error  string `json:"error,omitempty"`
}

type RetrieveRequest struct {
    Query string `json:"query"`
    Limit int    `json:"limit,omitempty"`
}

type RetrieveResponse struct {
    Results []string `json:"results,omitempty"`
    Status  string   `json:"status"`
    Error   string   `json:"error,omitempty"`
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
    var req SaveRequest
    var res SaveResponse

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        res.Status = "error"
        res.Error = "Invalid request format"
        json.NewEncoder(w).Encode(res)
        return
    }

    summary, err := summ.Summarize(req.Text)
    if err != nil {
        res.Status = "error"
        res.Error = err.Error()
        json.NewEncoder(w).Encode(res)
        return
    }

    embedding, err := emb.CreateEmbedding(summary)
    if err != nil {
        res.Status = "error"
        res.Error = err.Error()
        json.NewEncoder(w).Encode(res)
        return
    }

    embeddingBytes, err := vector.Float32SliceToBytes(embedding)
    if err != nil {
        res.Status = "error"
        res.Error = err.Error()
        json.NewEncoder(w).Encode(res)
        return
    }

    id := projectmemory.GenerateHash(summary, time.Now().UnixNano())
    err = store.Store(id, summary, embeddingBytes, time.Now())
    if err != nil {
        res.Status = "error"
        res.Error = err.Error()
        json.NewEncoder(w).Encode(res)
        return
    }

    res.Status = "success"
    res.ID = id
    json.NewEncoder(w).Encode(res)
}

func retrieveHandler(w http.ResponseWriter, r *http.Request) {
    var req RetrieveRequest
    var res RetrieveResponse

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        res.Status = "error"
        res.Error = "Invalid request format"
        json.NewEncoder(w).Encode(res)
        return
    }

    limit := req.Limit
    if limit <= 0 {
        limit = 5
    }

    queryEmbedding, err := emb.CreateEmbedding(req.Query)
    if err != nil {
        res.Status = "error"
        res.Error = err.Error()
        json.NewEncoder(w).Encode(res)
        return
    }

    results, err := store.Search(queryEmbedding, limit)
    if err != nil {
        res.Status = "error"
        res.Error = err.Error()
        json.NewEncoder(w).Encode(res)
        return
    }

    res.Status = "success"
    res.Results = results
    json.NewEncoder(w).Encode(res)
}

func main() {
    if err := initComponents(); err != nil {
        panic(err)
    }
    defer store.Close()

    r := mux.NewRouter()
    r.HandleFunc("/memory/save", saveHandler).Methods("POST")
    r.HandleFunc("/memory/retrieve", retrieveHandler).Methods("POST")

    http.ListenAndServe(":8080", r)
}
```

### Integrating with an MCP Server

```go
package main

import (
    "time"

    "github.com/localrivet/gomcp"
    gomcpserver "github.com/localrivet/gomcp/server"

    "github.com/localrivet/project-memory"
    "github.com/localrivet/project-memory/internal/contextstore"
    "github.com/localrivet/project-memory/internal/summarizer"
    "github.com/localrivet/project-memory/internal/tools"
    "github.com/localrivet/project-memory/internal/vector"
)

func main() {
    // Initialize Project-Memory components
    store := contextstore.NewSQLiteContextStore()
    store.Initialize(".memory.db")
    defer store.Close()

    summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
    summ.Initialize()

    emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
    emb.Initialize()

    // Create your MCP server
    mcpServer := gomcp.NewServer("your-app-name")

    // Add your own tools
    mcpServer = mcpServer.Tool("your_tool", "Your custom tool description",
        func(ctx *gomcpserver.Context, req YourRequest) (YourResponse, error) {
            // Your tool implementation
            return YourResponse{}, nil
        })

    // Add Project-Memory tools
    mcpServer = mcpServer.Tool(tools.ToolSaveContext, "Save context to memory",
        func(ctx *gomcpserver.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
            response := tools.SaveContextResponse{
                Status: "success",
            }

            summary, err := summ.Summarize(req.ContextText)
            if err != nil {
                response.Status = "error"
                response.Error = err.Error()
                return response, nil
            }

            embedding, err := emb.CreateEmbedding(summary)
            if err != nil {
                response.Status = "error"
                response.Error = err.Error()
                return response, nil
            }

            embeddingBytes, err := vector.Float32SliceToBytes(embedding)
            if err != nil {
                response.Status = "error"
                response.Error = err.Error()
                return response, nil
            }

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

    // Add retrieve context tool
    mcpServer = mcpServer.Tool(tools.ToolRetrieveContext, "Retrieve relevant context",
        func(ctx *gomcpserver.Context, req tools.RetrieveContextRequest) (tools.RetrieveContextResponse, error) {
            response := tools.RetrieveContextResponse{
                Status: "success",
            }

            queryEmbedding, err := emb.CreateEmbedding(req.Query)
            if err != nil {
                response.Status = "error"
                response.Error = err.Error()
                return response, nil
            }

            limit := req.Limit
            if limit <= 0 {
                limit = tools.DefaultRetrieveLimit
            }

            results, err := store.Search(queryEmbedding, limit)
            if err != nil {
                response.Status = "error"
                response.Error = err.Error()
                return response, nil
            }

            response.Results = results
            return response, nil
        })

    // Start your MCP server
    mcpServer.AsStdio().Run()
}

// Your custom request/response types
type YourRequest struct {
    // Fields...
}

type YourResponse struct {
    // Fields...
}
```

## Advanced Topics

### Custom Embedders

While Project-Memory provides a MockEmbedder for development, you'll want to use a real embedding provider in production.

Here's how to create a custom embedder:

```go
package main

import (
    "github.com/localrivet/project-memory/internal/vector"
)

// CustomEmbedder implements the vector.Embedder interface
type CustomEmbedder struct {
    dimension int
    // Add fields for API clients, configuration, etc.
}

// NewCustomEmbedder creates a new instance of CustomEmbedder
func NewCustomEmbedder(dimension int) *CustomEmbedder {
    return &CustomEmbedder{
        dimension: dimension,
    }
}

// Initialize sets up the embedder
func (e *CustomEmbedder) Initialize() error {
    // Set up API clients, load models, etc.
    return nil
}

// CreateEmbedding generates an embedding for the given text
func (e *CustomEmbedder) CreateEmbedding(text string) ([]float32, error) {
    // Call your embedding API or local model
    // For example:
    // response, err := callEmbeddingAPI(text)
    // if err != nil {
    //     return nil, err
    // }
    // return convertToFloat32Slice(response.Embedding), nil

    // Mock implementation for example purposes:
    embedding := make([]float32, e.dimension)
    for i := 0; i < e.dimension; i++ {
        embedding[i] = 0.1 // Replace with actual embedding values
    }
    return embedding, nil
}
```

Then use it in your application:

```go
embedder := NewCustomEmbedder(1536)
embedder.Initialize()
// Now use it like any other embedder
```

### Custom Summarizers

Similarly, you can create custom summarizers:

```go
package main

import (
    "github.com/localrivet/project-memory/internal/summarizer"
)

// CustomSummarizer implements the summarizer.Summarizer interface
type CustomSummarizer struct {
    maxLength int
    // Add fields for API clients, configuration, etc.
}

// NewCustomSummarizer creates a new instance of CustomSummarizer
func NewCustomSummarizer(maxLength int) *CustomSummarizer {
    return &CustomSummarizer{
        maxLength: maxLength,
    }
}

// Initialize sets up the summarizer
func (s *CustomSummarizer) Initialize() error {
    // Set up API clients, load models, etc.
    return nil
}

// Summarize generates a summary of the given text
func (s *CustomSummarizer) Summarize(text string) (string, error) {
    // Call your summarization API or local model
    // For example:
    // response, err := callSummarizationAPI(text, s.maxLength)
    // if err != nil {
    //     return "", err
    // }
    // return response.Summary, nil

    // Mock implementation for example purposes:
    if len(text) <= s.maxLength {
        return text, nil
    }
    return text[:s.maxLength] + "...", nil
}
```

### Database Maintenance

For long-running applications, you might want to add database maintenance:

```go
package main

import (
    "time"

    "github.com/localrivet/project-memory/internal/contextstore"
)

// This example assumes a SQLite context store with additional maintenance methods

// CleanupOldEntries removes entries older than the specified duration
func CleanupOldEntries(store contextstore.ContextStore, age time.Duration) error {
    sqlStore, ok := store.(*contextstore.SQLiteContextStore)
    if !ok {
        return fmt.Errorf("store is not a SQLiteContextStore")
    }

    // Execute maintenance SQL
    // In a real implementation, you'd add methods to the SQLiteContextStore
    // for operations like this

    return nil
}

// Compact optimizes the database
func CompactDatabase(store contextstore.ContextStore) error {
    sqlStore, ok := store.(*contextstore.SQLiteContextStore)
    if !ok {
        return fmt.Errorf("store is not a SQLiteContextStore")
    }

    // Execute VACUUM or similar operations

    return nil
}

// ScheduleMaintenance sets up periodic maintenance
func ScheduleMaintenance(store contextstore.ContextStore) {
    ticker := time.NewTicker(24 * time.Hour)
    go func() {
        for range ticker.C {
            // Remove entries older than 30 days
            CleanupOldEntries(store, 30*24*time.Hour)

            // Compact database
            CompactDatabase(store)
        }
    }()
}
```

## Examples

### Simple Save and Retrieve

```go
package main

import (
    "fmt"
    "time"

    "github.com/localrivet/project-memory"
    "github.com/localrivet/project-memory/internal/contextstore"
    "github.com/localrivet/project-memory/internal/summarizer"
    "github.com/localrivet/project-memory/internal/vector"
)

func main() {
    // Initialize components
    store := contextstore.NewSQLiteContextStore()
    store.Initialize(".memory.db")
    defer store.Close()

    summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
    summ.Initialize()

    emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
    emb.Initialize()

    // Sample texts
    texts := []string{
        "The quarterly meeting is scheduled for next Tuesday at 2 PM.",
        "Remember to submit the Q3 financial report by the end of the month.",
        "The new product launch is planned for October 15.",
        "Contact the marketing team to finalize the press release.",
        "The client feedback survey should be sent out next week.",
    }

    // Store all texts
    for _, text := range texts {
        summary, _ := summ.Summarize(text)
        embedding, _ := emb.CreateEmbedding(summary)
        embeddingBytes, _ := vector.Float32SliceToBytes(embedding)
        id := projectmemory.GenerateHash(summary, time.Now().UnixNano())
        store.Store(id, summary, embeddingBytes, time.Now())
        fmt.Printf("Stored: %s\n", text)
    }

    // Retrieve with different queries
    queries := []string{
        "meeting",
        "financial report",
        "product launch",
        "marketing",
        "survey",
    }

    for _, query := range queries {
        fmt.Printf("\nQuery: %s\n", query)
        queryEmbedding, _ := emb.CreateEmbedding(query)
        results, _ := store.Search(queryEmbedding, 2)
        for i, result := range results {
            fmt.Printf("  %d: %s\n", i+1, result)
        }
    }
}
```

### Using the Higher-Level API

```go
package main

import (
    "fmt"

    "github.com/localrivet/project-memory"
)

func main() {
    config := projectmemory.DefaultConfig()
    config.Database.Path = ".memory.db"

    server, err := projectmemory.NewServer(config)
    if err != nil {
        fmt.Printf("Error creating server: %v\n", err)
        return
    }

    // Save context examples
    texts := []string{
        "The quarterly meeting is scheduled for next Tuesday at 2 PM.",
        "Remember to submit the Q3 financial report by the end of the month.",
        "The new product launch is planned for October 15.",
    }

    for _, text := range texts {
        id, err := server.SaveContext(text)
        if err != nil {
            fmt.Printf("Error saving context: %v\n", err)
            continue
        }
        fmt.Printf("Saved text with ID: %s\n", id)
    }

    // Retrieve context
    queries := []string{"meeting", "financial", "product"}

    for _, query := range queries {
        fmt.Printf("\nQuery: %s\n", query)
        results, err := server.RetrieveContext(query, 2)
        if err != nil {
            fmt.Printf("Error retrieving context: %v\n", err)
            continue
        }

        for i, result := range results {
            fmt.Printf("  %d: %s\n", i+1, result)
        }
    }
}
```

## Troubleshooting

### Common Issues and Solutions

#### "Failed to initialize SQLite context store"

This usually means the SQLite database couldn't be created or opened. Check:

- Does the directory exist?
- Do you have write permissions?
- Is SQLite installed?

Solution: Specify an absolute path or ensure the directory exists.

#### "Failed to initialize summarizer"

This can happen if the provider configuration is incorrect. Check:

- Are you using the correct provider name?
- Do you have the necessary API keys set?

Solution: Use the mock provider during development or check your API key configuration.

#### "Failed to create embedding"

If using a real embedder, this could indicate API issues. Check:

- Is your internet connection working?
- Is the API key valid?
- Are you hitting rate limits?

Solution: Fall back to the mock embedder for testing or retry with exponential backoff.

#### "Error in vector search"

This could indicate database corruption or compatibility issues. Check:

- Is the database file intact?
- Are you using compatible embeddings?

Solution: Try recreating the database or check embedding dimensions.

### Getting Help

If you encounter issues not covered in this guide:

1. Check the [GitHub repository](https://github.com/localrivet/project-memory/issues) for existing issues
2. Review the logs for detailed error messages
3. Create a new issue with detailed reproduction steps

## Final Tips

- **Memory Management**: For large datasets, monitor memory usage and consider periodic cleanup
- **Error Handling**: Always check errors from all component operations
- **Graceful Shutdown**: Ensure you close the store properly to avoid data corruption
- **Testing**: Create mock implementations for testing your integration without external dependencies
- **Security**: Keep your database file secure and use environment variables for API keys
