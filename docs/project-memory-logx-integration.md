# Project-Memory Logging Integration Requirements

## Overview

We need project-memory to use the standard Go `log/slog` package in a way that allows importing projects to control all logging output in MCP mode, ensuring no logs corrupt the stdio JSON-RPC transport.

## Key Requirements

1. **Accept a pre-configured slog.Logger:**

   - Project-memory should accept a `*slog.Logger` in its initialization
   - This logger should be used throughout project-memory for all logging
   - Example API: `NewMemoryManager(storagePath string, logger *slog.Logger) (*MemoryManager, error)`

2. **Pass the logger to gomcp:**

   - When project-memory initializes its own gomcp components:
     ```go
     // Any gomcp server instances must use the provided logger
     srv := server.NewServer("projectmemory")
     srv.WithLogger(logger) // Use same logger passed from importing projects
     ```

3. **No direct stdout/stderr usage:**

   - Never use fmt.Print*/log.Print* directly
   - Always use the provided slog.Logger methods:

     ```go
     // DO NOT use:
     fmt.Println("Initializing memory")
     log.Printf("Error: %v", err)

     // INSTEAD use:
     logger.Info("Initializing memory")
     logger.Error("Error initializing", "error", err)
     ```

4. **Log level respect:**

   - Never override the log level set by the parent application
   - The logger will already have the appropriate level set

## Implementation Example

```go
package projectmemory

import (
  "log/slog"
  "github.com/localrivet/gomcp/server"
)

type MemoryManager struct {
  logger *slog.Logger
  // other fields...
}

func NewMemoryManager(storagePath string, logger *slog.Logger) (*MemoryManager, error) {
  if logger == nil {
    // Default logger only if none provided
    logger = slog.Default()
  }

  manager := &MemoryManager{
    logger: logger,
    // initialize other fields...
  }

  // Log using the provided logger
  manager.logger.Info("Initializing memory manager", "path", storagePath)

  // When creating MCP components, use the same logger
  mcpServer := server.NewServer("memory-server")
  mcpServer.WithLogger(logger)

  return manager, nil
}

// Methods always use the stored logger
func (m *MemoryManager) DoSomething() error {
  m.logger.Debug("Doing something...")
  // implementation...
  return nil
}
```

## Critical Points

- In MCP mode, importing projects redirect all logging to a file
- ANY output to stdout/stderr will corrupt the JSON communication
- The logger MUST be shared between components to ensure consistent behavior
- Log level is set to "error" in MCP mode to minimize output
