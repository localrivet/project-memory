# Project-Memory

[![Go Reference](https://pkg.go.dev/badge/github.com/localrivet/project-memory.svg)](https://pkg.go.dev/github.com/localrivet/project-memory)
[![Go Report Card](https://goreportcard.com/badge/github.com/localrivet/project-memory)](https://goreportcard.com/report/github.com/localrivet/project-memory)

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

Project-Memory can be used as a library in your Go applications in multiple ways:

1. **Direct Component Usage** - Directly use the core components for maximum control
2. **Helper Functions** - Use the `CreateComponents` helper for easier initialization
3. **High-Level API** - Use the Server API for simplified operations

These approaches allow you to integrate Project-Memory with your existing MCP server without conflicts. For detailed instructions and examples, see our [Library Usage Guide](docs/library_usage.md) and our comprehensive [Embedding Guide](docs/embedding_guide.md).

### Quick Example

```go
import (
    "github.com/localrivet/project-memory"
    "github.com/localrivet/project-memory/internal/contextstore"
    "github.com/localrivet/project-memory/internal/summarizer"
    "github.com/localrivet/project-memory/internal/vector"
)

// Initialize components
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
