# Using ProjectMemory as a Library

This guide explains how to integrate ProjectMemory as a library in your application, particularly when you already have your own MCP server.

## Integration Options

There are three main approaches to integrating ProjectMemory in your application, each with different levels of abstraction:

### Option 1: Direct Component Usage (Recommended)

This approach gives you the most control and avoids any MCP server conflicts. You directly initialize and use the core components of ProjectMemory.

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

    // To delete a specific context entry:
    err := store.DeleteContext(id)
    if err != nil {
        // Handle error
    }

    // To replace an existing context entry:
    updatedText := "This is updated context information."
    updatedSummary, _ := summ.Summarize(updatedText)
    updatedEmbedding, _ := emb.CreateEmbedding(updatedSummary)
    updatedEmbeddingBytes, _ := vector.Float32SliceToBytes(updatedEmbedding)
    store.ReplaceContext(id, updatedSummary, updatedEmbeddingBytes, time.Now())

    // To clear all context entries:
    err = store.ClearAllContext()
    if err != nil {
        // Handle error
    }

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

    // Use the new memory management methods
    err = pmServer.DeleteContext(id)
    if err != nil {
        // Handle error
    }

    err = pmServer.ClearAllContext()
    if err != nil {
        // Handle error
    }

    err = pmServer.ReplaceContext(id, "This is replacement context")
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

When you have your own MCP server, you can integrate ProjectMemory's functionality by registering new tools that use ProjectMemory's components.

```go
import (
    "time"

    "github.com/localrivet/gomcp"
    gomcpserver "github.com/localrivet/gomcp/server"

    "github.com/localrivet/projectmemory"
    "github.com/localrivet/projectmemory/internal/tools"
)

func main() {
    // Initialize ProjectMemory components using any of the approaches above
    // ...

    // Create your own MCP server
    mcpServer := gomcp.NewServer("your-mcp-server")

    // Register ProjectMemory tools with your server
    mcpServer = mcpServer.Tool(tools.ToolSaveContext, "Save context to the memory store",
        func(ctx *gomcpserver.Context, req tools.SaveContextRequest) (tools.SaveContextResponse, error) {
            response := tools.SaveContextResponse{
                Status: "success",
            }

            // Use ProjectMemory components to implement the tool
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

    // Register delete context tool
    mcpServer = mcpServer.Tool(tools.ToolDeleteContext, "Delete a specific context entry",
        func(ctx *gomcpserver.Context, req tools.DeleteContextRequest) (tools.DeleteContextResponse, error) {
            response := tools.DeleteContextResponse{
                Status: "success",
            }

            // Delete the context entry
            err := store.DeleteContext(req.ID)
            if err != nil {
                response.Status = "error"
                response.Error = err.Error()
                return response, nil
            }

            return response, nil
        })

    // Register clear all context tool
    mcpServer = mcpServer.Tool(tools.ToolClearAllContext, "Clear all context entries",
        func(ctx *gomcpserver.Context, req tools.ClearAllContextRequest) (tools.ClearAllContextResponse, error) {
            response := tools.ClearAllContextResponse{
                Status: "success",
            }

            // Verify confirmation
            if req.Confirmation != "confirm" {
                response.Status = "error"
                response.Error = "Confirmation required. Set 'confirmation' to 'confirm' to proceed with clearing all context."
                return response, nil
            }

            // Clear all context entries
            err := store.ClearAllContext()
            if err != nil {
                response.Status = "error"
                response.Error = err.Error()
                return response, nil
            }

            return response, nil
        })

    // Register replace context tool
    mcpServer = mcpServer.Tool(tools.ToolReplaceContext, "Replace an existing context entry",
        func(ctx *gomcpserver.Context, req tools.ReplaceContextRequest) (tools.ReplaceContextResponse, error) {
            response := tools.ReplaceContextResponse{
                Status: "success",
            }

            // Process the new context text
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

            // Replace the context entry
            err = store.ReplaceContext(req.ID, summary, embeddingBytes, time.Now())
            if err != nil {
                response.Status = "error"
                response.Error = err.Error()
                return response, nil
            }

            return response, nil
        })

    // Start your MCP server
    mcpServer.AsStdio().Run()
}
```

## Best Practices

1. **Prefer Direct Component Usage**: For most integrations, direct component usage (Option 1) is recommended as it gives you the most control and avoids any MCP server conflicts.

2. **Close Resources**: Always close resources properly, especially the context store, by using `defer store.Close()`.

3. **Error Handling**: Handle errors from each component initialization and operation to ensure reliability.

4. **Avoid Running Multiple MCP Servers**: Never run both your own MCP server and ProjectMemory's server at the same time, as they would compete for the same resources.

5. **Configuration Management**: Use the `DefaultConfig()` function for sensible defaults and override only what you need.

6. **Memory Management**: Regularly maintain your context store by:
   - Deleting obsolete context entries that are no longer needed
   - Replacing outdated information with updated content
   - Considering a periodic cleanup strategy based on your application's needs

## Complete Example

See the [embed-in-mcp example](../examples/embed-in-mcp/main.go) for a complete working example of integrating ProjectMemory with an existing MCP server.
