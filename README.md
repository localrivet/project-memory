# Project-Memory

Project-Memory is an MCP (Model Context Protocol) server that provides persistent storage for conversation context information using SQLite. This allows LLMs to remember and retrieve relevant information from past interactions.

## Overview

Project-Memory implements two MCP tools:

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
  "models": {
    "provider": "mock",
    "modelId": "mock-model",
    "maxTokens": 1024,
    "temperature": 0.7
  },
  "database": {
    "path": ".projectmemory.db"
  },
  "logging": {
    "level": "INFO",
    "format": "TEXT"
  }
}
```

### Running the Server

From the project root:

```sh
go run cmd/project-memory/main.go
```

## Using as a Library

Project-Memory can be embedded as a library in another MCP server. This allows you to add persistent context capabilities to your existing MCP applications.

### Example: Embedding in Another MCP Server

See the `examples/embed-in-mcp/main.go` file for a complete example.

```go
import (
    "github.com/localrivet/project-memory/internal/contextstore"
    "github.com/localrivet/project-memory/internal/summarizer"
    "github.com/localrivet/project-memory/internal/vector"
    "github.com/localrivet/project-memory/internal/tools"
)

// Initialize components
store := contextstore.NewSQLiteContextStore()
store.Initialize(".projectmemory.db")

summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
summ.Initialize()

emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
emb.Initialize()

// Create your own MCP server
mcpServer := gomcp.NewServer("your-server")

// Add Project-Memory tools to your server
mcpServer = mcpServer.Tool(tools.ToolSaveContext, "Save context",
    func(ctx *server.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
        // Implementation using store, summ, and emb
    })

mcpServer = mcpServer.Tool(tools.ToolRetrieveContext, "Retrieve context",
    func(ctx *server.Context, req tools.RetrieveContextRequest) (tools.RetrieveContextResponse, error) {
        // Implementation using store, summ, and emb
    })
```

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

## License

[MIT License](LICENSE)
# project-memory
