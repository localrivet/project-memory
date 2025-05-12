# Using Project-Memory as a Library

This guide explains how to integrate Project-Memory as a library in your application, particularly when you already have your own MCP server.

## Integration Options

There are three main approaches to integrating Project-Memory in your application, each with different levels of abstraction:

### Option 1: Direct Component Usage (Recommended)

This approach gives you the most control and avoids any MCP server conflicts. You directly initialize and use the core components of Project-Memory.

```go
import (
    "time"

    "github.com/localrivet/projectmemory"
    "github.com/localrivet/projectmemory/internal/contextstore"
    "github.com/localrivet/projectmemory/internal/summarizer"
    "github.com/localrivet/projectmemory/internal/vector"
)

func main() {
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

    // To retrieve context:
    queryText := "test context"
    queryEmbedding, _ := emb.CreateEmbedding(queryText)
    results, _ := store.Search(queryEmbedding, 5)

    // Your existing application code continues...
}
```

### Option 2: Using CreateComponents Helper

This approach uses a helper function to create all the components at once, which is slightly more convenient but still gives you direct access to the components.

```go
import (
    "github.com/localrivet/projectmemory"
)

func main() {
    // Create a configuration
    config := projectmemory.DefaultConfig()
    config.Database.Path = ".projectmemory.db"

    // Initialize all components at once
    store, summ, emb, err := projectmemory.CreateComponents(config)
    if err != nil {
        // Handle error
    }
    defer store.Close()

    // Use components directly as in Option 1
    // ...

    // Your existing application code continues...
}
```

### Option 3: Server API (Without Starting)

This approach uses the high-level Server API but avoids starting the MCP server. This is useful if you want the convenience of higher-level functions but still need to integrate with your own MCP server.

```go
import (
    "github.com/localrivet/projectmemory"
)

func main() {
    // Create and initialize server
    config := projectmemory.DefaultConfig()
    pmServer, err := projectmemory.NewServer(config)
    if err != nil {
        // Handle error
    }

    // You can use the high-level methods
    id, err := pmServer.SaveContext("This is a test context")
    if err != nil {
        // Handle error
    }

    results, err := pmServer.RetrieveContext("test", 5)
    if err != nil {
        // Handle error
    }

    // Or access the components directly if needed
    store := pmServer.GetStore()
    summ := pmServer.GetSummarizer()
    emb := pmServer.GetEmbedder()

    // Important: Do NOT call pmServer.Start()
    // Instead, continue with your own MCP server initialization

    // Your existing application code continues...
}
```

## Integrating with Your MCP Server

When you have your own MCP server, you can integrate Project-Memory's functionality by registering new tools that use Project-Memory's components.

```go
import (
    "time"

    "github.com/localrivet/gomcp"
    gomcpserver "github.com/localrivet/gomcp/server"

    "github.com/localrivet/projectmemory"
    "github.com/localrivet/projectmemory/internal/tools"
)

func main() {
    // Initialize Project-Memory components using any of the approaches above
    // ...

    // Create your own MCP server
    mcpServer := gomcp.NewServer("your-mcp-server")

    // Register Project-Memory tools with your server
    mcpServer = mcpServer.Tool(tools.ToolSaveContext, "Save context to the memory store",
        func(ctx *gomcpserver.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
            response := tools.SaveContextResponse{
                Status: "success",
            }

            // Use Project-Memory components to implement the tool
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

    // Similarly for retrieve context...

    // Start your MCP server
    mcpServer.AsStdio().Run()
}
```

## Best Practices

1. **Prefer Direct Component Usage**: For most integrations, direct component usage (Option 1) is recommended as it gives you the most control and avoids any MCP server conflicts.

2. **Close Resources**: Always close resources properly, especially the context store, by using `defer store.Close()`.

3. **Error Handling**: Handle errors from each component initialization and operation to ensure reliability.

4. **Avoid Running Multiple MCP Servers**: Never run both your own MCP server and Project-Memory's server at the same time, as they would compete for the same resources.

5. **Configuration Management**: Use the `DefaultConfig()` function for sensible defaults and override only what you need.

## Complete Example

See the [embed-in-mcp example](../examples/embed-in-mcp/main.go) for a complete working example of integrating Project-Memory with an existing MCP server.
