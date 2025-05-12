# Development Guide

This guide provides information for developers who want to contribute to Project-Memory or integrate it as a library in their applications.

## Setting Up the Development Environment

### Prerequisites

- Go 1.20+
- SQLite
- Git

### Getting the Source Code

```bash
git clone https://github.com/localrivet/projectmemory.git
cd project-memory
go mod download
```

## Project Structure

The Project-Memory codebase is organized as follows:

```
project-memory/
├── cmd/                  # Application entry points
│   └── project-memory/   # Main server application
├── docs/                 # Documentation
├── examples/             # Example applications
│   └── embed-in-mcp/     # Example of embedding in another MCP server
├── internal/             # Internal packages
│   ├── contextstore/     # SQLite context storage
│   ├── logger/           # Structured logging
│   ├── server/           # MCP server implementation
│   ├── summarizer/       # Text summarization
│   │   └── providers/    # AI provider implementations
│   ├── telemetry/        # Performance metrics
│   ├── tools/            # MCP tool schemas
│   └── vector/           # Vector operations and embedding
└── scripts/              # Utility scripts
```

## Key Components

### ContextStore

The `contextstore` package provides interfaces and implementations for storing and retrieving context entries.

**Key Interfaces:**

```go
// ContextStore defines the interface for storing and retrieving context
type ContextStore interface {
    Initialize(dbPath string) error
    Store(id, text string, embedding []byte, timestamp time.Time) error
    Search(queryEmbedding []float32, limit int) ([]string, error)
    Close() error
}
```

The default implementation is `SQLiteContextStore`, which uses SQLite for persistence.

### Summarizer

The `summarizer` package handles text summarization using various AI providers.

**Key Interfaces:**

```go
// Summarizer defines the interface for text summarization
type Summarizer interface {
    Initialize() error
    Summarize(text string) (string, error)
}
```

The default implementation is `BasicSummarizer`, which uses AI providers to generate concise summaries.

### Vector

The `vector` package provides utilities for working with embeddings and vector operations.

**Key Interfaces:**

```go
// Embedder defines the interface for creating text embeddings
type Embedder interface {
    Initialize() error
    CreateEmbedding(text string) ([]float32, error)
}
```

A mock implementation, `MockEmbedder`, is provided for testing and development. In production, you would typically use a real embedding model.

## Embedding as a Library

Project-Memory can be used as a library in your applications. Please refer to our comprehensive [Library Usage Guide](library_usage.md) for detailed information on the various integration options.

The Library Usage Guide covers:

- Direct component usage (recommended approach)
- Using the CreateComponents helper function
- Using the Server API without starting the MCP server
- Integrating with your own MCP server
- Best practices for library integration

For a quick example of how to use Project-Memory components directly in your application, see the example below:

```go
// Initialize store
store := contextstore.NewSQLiteContextStore()
if err := store.Initialize("path/to/db.sqlite"); err != nil {
    log.Fatalf("Failed to initialize store: %v", err)
}
defer store.Close()

// Initialize summarizer
summ := summarizer.NewBasicSummarizer(500) // Max summary length
if err := summ.Initialize(); err != nil {
    log.Fatalf("Failed to initialize summarizer: %v", err)
}

// Initialize embedder
emb := vector.NewMockEmbedder(1536) // Embedding dimension
if err := emb.Initialize(); err != nil {
    log.Fatalf("Failed to initialize embedder: %v", err)
}

// Now you can use these components directly
summary, _ := summ.Summarize("Your text here")
embedding, _ := emb.CreateEmbedding(summary)
// ...
```

For a complete example, see [examples/embed-in-mcp/main.go](../examples/embed-in-mcp/main.go).

## Testing

Run the tests with:

```bash
go test ./...
```

For coverage reporting:

```bash
go test ./... -cover
```

## Contributing

Contributions are welcome! Here's how to contribute:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Ensure all tests pass
5. Submit a pull request

### Code Style

- Follow standard Go code style and best practices
- Use meaningful variable and function names
- Add comments for non-trivial code
- Write tests for new functionality

## Building for Production

To build the server binary:

```bash
go build -o project-memory cmd/project-memory/main.go
```

For a smaller binary with debugging symbols removed:

```bash
go build -ldflags="-s -w" -o project-memory cmd/project-memory/main.go
```

## Creating Custom Providers

To add a new AI provider for summarization:

1. Add the provider implementation in `internal/summarizer/providers/`
2. Register the provider in `internal/summarizer/providers/factory.go`
3. Add appropriate configuration options
4. Implement the required interfaces
