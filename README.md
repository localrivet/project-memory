# ProjectMemory

[![Go Reference](https://pkg.go.dev/badge/github.com/localrivet/projectmemory.svg)](https://pkg.go.dev/github.com/localrivet/projectmemory)
[![Go Report Card](https://goreportcard.com/badge/github.com/localrivet/projectmemory)](https://goreportcard.com/report/github.com/localrivet/projectmemory)

ProjectMemory is an MCP (Model Context Protocol) server that provides persistent storage for conversation context information using SQLite. This allows LLMs to remember and retrieve relevant information from past interactions.

## Overview

ProjectMemory implements two MCP tools:

1. `save_context` - Saves context (conversation snippets, inputs, outputs) to a persistent store
2. `retrieve_context` - Retrieves relevant context based on semantic search

The service handles:

- Summarizing text to extract key information
- Creating embeddings for semantic search
- Storing context with metadata in SQLite
- Searching for relevant context using vector similarity

## Components

- **Vector Package**: Utilities for embedding operations and vector similarity
- **Summarizer Package**: Text summarization capabilities
- **ContextStore Package**: SQLite-based persistent storage
- **Server Package**: MCP server implementation
- **Logger Package**: Structured logging system

## Getting Started

### Prerequisites

- Go 1.20+
- SQLite

### Configuration

Create a `.projectmemoryconfig` file in the project root:

```json
{
  "store": {
    "sqlite_path": ".projectmemory.db"
  },
  "summarizer": {
    "provider": "basic"
  },
  "embedder": {
    "provider": "mock",
    "dimensions": 768
  },
  "logging": {
    "level": "info",
    "format": "text"
  }
}
```

You can also use environment variables to override configuration values:

```bash
# Set the database path
export SQLITE_PATH=".custom-db.db"

# Set log level
export LOG_LEVEL="debug"
```

For a detailed explanation of all configuration options, see the [Configuration Reference](docs/configuration.md).

### Running the Server

From the project root:

```sh
go run cmd/project-memory/main.go
```

## Using as a Library

ProjectMemory can be used as a library in your Go applications in multiple ways:

1. **Direct Component Usage** - Directly use the core components for maximum control
2. **Helper Functions** - Use the `CreateComponents` helper for easier initialization
3. **High-Level API** - Use the Server API for simplified operations

These approaches allow you to integrate ProjectMemory with your existing MCP server without conflicts. For detailed instructions and examples, see our [Library Usage Guide](docs/library_usage.md) and our comprehensive [Embedding Guide](docs/embedding_guide.md).

### Custom Logging

When embedding ProjectMemory in your application, you can override the default logger:

```go
import (
    "github.com/localrivet/gomcp/logx"
    "github.com/localrivet/projectmemory"
)

func main() {
    // Create your custom logger
    logger := logx.NewLogger("debug")

    // Initialize the server with your custom logger
    server, err := projectmemory.NewServer(".projectmemoryconfig", logger)
    if err != nil {
        logger.Error("Failed to create server: %v", err)
    }

    // Alternatively, you can create a server first and then set the logger
    // server, err := projectmemory.NewServer(".projectmemoryconfig", nil)
    // if err != nil {
    //     // Handle error
    // }
    //
    // server.WithLogger(logger)

    // Now all ProjectMemory logs will be routed through your logger
    // Continue with your application...
}
```

### Quick Example

```go
import (
    "time"

    "github.com/localrivet/projectmemory"
    "github.com/localrivet/projectmemory/internal/contextstore"
    "github.com/localrivet/projectmemory/internal/summarizer"
    "github.com/localrivet/projectmemory/internal/vector"
)

// Option 1: Use the components directly
store := contextstore.NewSQLiteContextStore()
store.Initialize(".projectmemory.db")
defer store.Close()

summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
summ.Initialize()

emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
emb.Initialize()

// Now you can use these components directly in your code
// For example, to store context:
testText := "This is a test context to save."
summary, _ := summ.Summarize(testText)
embedding, _ := emb.CreateEmbedding(summary)
embeddingBytes, _ := vector.Float32SliceToBytes(embedding)
id := projectmemory.GenerateHash(summary, time.Now().UnixNano())
store.Store(id, summary, embeddingBytes, time.Now())

// Option 2: Use the high-level API with configuration
cfg := projectmemory.DefaultConfig()
cfg.Store.SQLitePath = ".custom-memory.db"
pmServer, err := projectmemory.NewServer(".projectmemoryconfig", nil)
if err != nil {
    // Handle error
}

// Save context using the high-level API
id, err = pmServer.SaveContext("This is a test context from the high-level API")
if err != nil {
    // Handle error
}
```

For a complete example of integrating with an existing MCP server, see `examples/embed-in-mcp/main.go`.

## API Reference

### Tool: save_context

**Request:**

```json
{
  "context_text": "The text to save in the context store"
}
```

**Response:**

```json
{
  "status": "success",
  "id": "generated-unique-id"
}
```

### Tool: retrieve_context

**Request:**

```json
{
  "query": "The query to search for",
  "limit": 5
}
```

**Response:**

```json
{
  "status": "success",
  "results": ["Matching context entry 1", "Matching context entry 2", "..."]
}
```

### Tool: delete_context

**Request:**

```json
{
  "id": "context-entry-id-to-delete"
}
```

**Response:**

```json
{
  "status": "success"
}
```

### Tool: clear_all_context

**Request:**

```json
{
  "confirmation": "confirm"
}
```

**Response:**

```json
{
  "status": "success"
}
```

### Tool: replace_context

**Request:**

```json
{
  "id": "context-entry-id-to-replace",
  "context_text": "The new text to replace the existing context"
}
```

**Response:**

```json
{
  "status": "success"
}
```

## Documentation

- [Installation Guide](docs/installation.md)
- [Configuration Reference](docs/configuration.md)
- [API Reference](docs/api.md)
- [Development Guide](docs/development.md)
- [Architecture Documentation](docs/architecture.md)
- [Library Usage Guide](docs/library_usage.md)
- [Embedding Guide](docs/embedding_guide.md) - Comprehensive guide for embedding in your application

## License

[MIT License](LICENSE)
