# Go's slog Package: A Structured Logging Guide

## Introduction

The `log/slog` package, introduced in Go 1.21, provides a modern, structured logging solution for Go applications. Unlike traditional logging systems that output free-form text messages, structured logging organizes log data into a standardized format with well-defined fields, making logs easier to search, filter, and analyze.

This guide explains how `slog` works and how to effectively use it in your applications.

## Core Concepts

### Structured Logging

In structured logging, each log entry consists of:

- A timestamp
- A severity level (like Info, Debug, Error)
- A message
- A collection of key-value pairs (attributes)

This approach makes logs machine-readable while preserving human readability.

### Key Components of slog

1. **Logger**: The main entry point for logging
2. **Handler**: Processes and outputs log records
3. **Record**: Contains all data for a single log event
4. **Level**: Indicates severity/importance
5. **Attr**: Represents a key-value pair (attribute)
6. **Value**: Type-safe container for attribute values

## Basic Usage

### Top-level Functions

The simplest way to use `slog` is through its top-level functions:

```go
import "log/slog"

func main() {
    // Simple logging with key-value pairs
    slog.Info("user logged in", "username", "alice", "user_id", 42)
    slog.Error("operation failed", "error", err, "duration", time.Since(start))

    // Debug and Warning levels
    slog.Debug("detailed information", "config", config)
    slog.Warn("resource usage high", "memory_mb", 950, "threshold_mb", 1000)
}
```

### Using Context

For improved observability and integration with tracing systems, `slog` supports passing a `context.Context`:

```go
func HandleRequest(ctx context.Context, req *Request) {
    // Use the context-aware variants
    slog.InfoContext(ctx, "processing request", "request_id", req.ID)

    // The context is propagated to handlers that can extract trace IDs, etc.
    if err := process(ctx, req); err != nil {
        slog.ErrorContext(ctx, "failed to process request",
            "request_id", req.ID, "error", err)
    }
}
```

## Controlling Log Output

### Configuring Handlers

`slog` provides two built-in handlers:

1. **TextHandler**: Outputs logs in a key=value format
2. **JSONHandler**: Outputs logs as line-delimited JSON

```go
// Text handler (machine-parsable but human-readable)
textHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
    AddSource: true,
})
logger := slog.New(textHandler)

// JSON handler (ideal for production environments)
jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})
jsonLogger := slog.New(jsonHandler)
```

### Setting the Default Logger

You can replace the default logger to affect all top-level function calls:

```go
// Make our configured logger the default
slog.SetDefault(logger)

// Now top-level functions use our logger
slog.Info("this uses our custom handler")
```

### Dynamic Log Levels

`slog` allows changing log levels at runtime using `LevelVar`:

```go
// Create a variable to hold the level
programLevel := new(slog.LevelVar) // Defaults to LevelInfo

// Use it when creating a handler
handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: programLevel,
})
logger := slog.New(handler)
slog.SetDefault(logger)

// Later, change the level dynamically
programLevel.Set(slog.LevelDebug) // Now Debug logs will appear
```

## Advanced Features

### Attribute Groups

You can organize related attributes into logical groups:

```go
// Create a group of attributes
slog.Info("request processed",
    slog.Group("user",
        "id", user.ID,
        "name", user.Name,
        "role", user.Role,
    ),
    slog.Group("request",
        "method", req.Method,
        "path", req.URL.Path,
        "duration_ms", duration.Milliseconds(),
    ),
)
```

In JSON output, this creates nested objects:

```json
{
  "time": "2023-08-15T15:53:54Z",
  "level": "INFO",
  "msg": "request processed",
  "user": {
    "id": 42,
    "name": "alice",
    "role": "admin"
  },
  "request": {
    "method": "GET",
    "path": "/api/resources",
    "duration_ms": 237
  }
}
```

### Logger with Predefined Attributes

Create loggers with predefined attributes that appear in every log entry:

```go
// Create a logger with request-specific attributes
requestLogger := logger.With(
    "request_id", requestID,
    "user_id", userID,
    "client_ip", clientIP,
)

// All logs from this logger include those attributes
requestLogger.Info("processing started")
requestLogger.Info("step 1 complete", "duration_ms", step1Time)
requestLogger.Info("processing complete", "status", "success")
```

### Attribute Construction

For more control over attribute types, use specific constructors:

```go
slog.Info("server stats",
    slog.Int("active_connections", activeCount),
    slog.Duration("uptime", time.Since(startTime)),
    slog.Time("started_at", startTime),
    slog.Bool("healthy", isHealthy),
    slog.Float64("load_average", loadAvg),
)
```

### Custom Value Logging with LogValuer

Implement `LogValuer` interface to control how types are logged:

```go
// Implement custom logging behavior for sensitive data
type Credentials struct {
    Username string
    Password string
}

// LogValue controls how Credentials appear in logs
func (c Credentials) LogValue() slog.Value {
    return slog.GroupValue(
        slog.String("username", c.Username),
        slog.String("password", "********"), // Redact password
    )
}

// Now credentials will be safely logged
creds := Credentials{Username: "admin", Password: "secret123"}
slog.Info("user credentials", "credentials", creds)
```

### Custom Handlers

For specialized logging needs, you can implement the `Handler` interface:

```go
type Handler interface {
    Enabled(context.Context, Level) bool
    Handle(context.Context, Record) error
    WithAttrs([]Attr) Handler
    WithGroup(string) Handler
}
```

Common use cases for custom handlers include:

- Sending logs to specialized services
- Adding custom filtering logic
- Reformatting logs for specific consumers
- Adding metadata like hostname or environment

## Performance Tips

### Avoid Unnecessary Computation

Use `LogValuer` to defer expensive operations:

```go
type ExpensiveValue struct {
    // Data needed to compute value
    param string
}

func (e ExpensiveValue) LogValue() slog.Value {
    // Only computed if the log level is enabled
    result := computeExpensiveValue(e.param)
    return slog.AnyValue(result)
}

// Now expensive computation only happens if needed
slog.Debug("detailed info", "data", ExpensiveValue{param: "input"})
```

### Use Precomputed Loggers

Create loggers with common attributes in advance:

```go
var (
    serverLogger = slog.With("component", "server", "version", version)
    dbLogger     = slog.With("component", "database")
    apiLogger    = slog.With("component", "api")
)

func StartServer() {
    serverLogger.Info("server starting", "port", config.Port)
    // ...
}
```

### Use LogAttrs for Maximum Efficiency

The most efficient way to log is using `LogAttrs`:

```go
// Most efficient logging (avoids allocations)
logger.LogAttrs(ctx, slog.LevelInfo, "message",
    slog.String("key1", "value1"),
    slog.Int("key2", 42))
```

## Integration with Existing Code

### Standard Library Integration

`slog` integrates with the standard `log` package:

```go
// Create a standard log.Logger that uses slog
stdLogger := slog.NewLogLogger(handler, slog.LevelInfo)

// Third-party libraries using the standard logger will now use slog
log.SetOutput(stdLogger.Writer())
```

### Compatible with io.Writer

You can integrate `slog` with anything that accepts an `io.Writer`:

```go
// Create a writer that logs at error level
logWriter := slog.NewLogLogger(slog.Default().Handler(), slog.LevelError)

// Use it with a library expecting io.Writer
thirdPartyLib.SetErrorOutput(logWriter.Writer())
```

## MCP Integration for Disabling Output

When running in MCP mode through stdio, it's critical to prevent any logs from contaminating the JSON communication channel. You can configure `slog` to discard all output:

```go
// Discard all logs when running as MCP stdio service
if strings.ToLower(os.Getenv("PROJECTMEMORY_LOG_OUTPUT")) == "discard" {
    outputWriter = io.Discard
    // Last log before switching to discard mode
    slog.Info("Logging disabled for MCP stdio mode. All log output will be discarded.")
} else {
    outputWriter = os.Stderr
}

// Configure with the appropriate writer
opts := &slog.HandlerOptions{
    AddSource: true,
    Level:     programLevel,
}
handler := slog.NewTextHandler(outputWriter, opts)
slog.SetDefault(slog.New(handler))
```

This pattern ensures logs won't interfere with the MCP protocol while still allowing rich logging in other execution contexts.

## Conclusion

Go's `slog` package offers a modern, flexible logging system that strikes a balance between simplicity and power. With structured logs, you get both human readability and machine parseability, making it easier to troubleshoot issues and analyze application behavior.

By understanding the core concepts and following the patterns outlined in this guide, you can effectively integrate `slog` into your Go applications and improve your observability infrastructure.
