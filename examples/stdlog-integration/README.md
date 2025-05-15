# Standard Library Logger Integration

This example demonstrates how to integrate the Go standard library's `log` package with ProjectMemory's structured logger. This allows you to:

1. Replace the global standard logger with a ProjectMemory structured logger
2. Create proxy standard loggers that route to ProjectMemory's logger
3. Mix usage of standard library logging and structured logging

## Key Features

### Global Logger Replacement

You can completely replace Go's global standard logger with a ProjectMemory structured logger, affecting all code in your application that uses the standard log package:

```go
// Configure your structured logger with desired features
structuredLogger := logger.New(&logger.Config{
    Level:  logger.INFO,
    Format: logger.JSON,              // Use JSON formatting
    Output: os.Stdout,                // Output to console
    DefaultTags: map[string]interface{}{
        "service": "my-service",
    },
})

// Replace the global standard logger
// All log.* calls will now use this structured logger process-wide
logger.ReplaceStdLogger(structuredLogger, logger.INFO)

// Standard library calls now use structured logging
log.Println("This message uses structured logging")
log.Printf("User %s logged in from %s", username, ipAddress)
```

### Proxy Loggers

For cases where you need to pass a `*log.Logger` to a library or component but want the output to go through your structured logging system:

```go
// Create a logger for a specific component
componentLogger := structuredLogger.WithField("component", "database")

// Create a standard library logger that proxies to your structured logger
stdLog := logger.CreateStdLogProxy(componentLogger, logger.WARN)

// Pass to components that require a standard library logger
someLibrary.Configure(stdLog)
```

### How It Works

The integration works by:

1. Creating an adapter that implements Go's `io.Writer` interface
2. This adapter captures log output and redirects it to your structured logger
3. The standard library's logger is reconfigured to use this adapter
4. All log calls automatically flow through your structured logger

## Running the Example

To run this example:

```bash
go run examples/stdlog-integration/main.go
```

## Practical Use Cases

- Gradually migrating large codebases to structured logging
- Working with third-party libraries that use the standard logger
- Maintaining backward compatibility while improving logging capabilities
- Ensuring all logs (including from dependencies) follow your logging format

## Related Documentation

See the [ProjectMemory Library Usage documentation](../../docs/library_usage.md#standard-logger-integration) for more details on standard logger integration.
