# Standard Library Integration

ProjectMemory's logger package provides powerful mechanisms to override the standard library's log package, allowing you to maintain compatibility with existing code while leveraging advanced structured logging features.

## API Reference

### Functions

#### `ReplaceStdLogger`

```go
func ReplaceStdLogger(logger *Logger, level LogLevel)
```

Replaces Go's global standard logger with a ProjectMemory structured logger, affecting all code in your application that uses the standard log package.

**Parameters:**

- `logger`: The ProjectMemory structured logger to use
- `level`: The log level at which standard library logs should be recorded

**Example:**

```go
// Create and configure a structured logger
structuredLogger := logger.New(&logger.Config{
    Level:  logger.INFO,
    Format: logger.JSON,
    Output: os.Stdout,
})

// Replace the global standard logger
logger.ReplaceStdLogger(structuredLogger, logger.INFO)

// All standard log calls now use the structured logger
log.Println("This uses structured logging")
```

#### `GetStdLogAdapter`

```go
func GetStdLogAdapter(logger *Logger, level LogLevel) io.Writer
```

Returns an `io.Writer` adapter that can be used to capture logs and redirect them to a ProjectMemory structured logger at the specified level.

**Parameters:**

- `logger`: The ProjectMemory structured logger to use
- `level`: The log level at which captured logs should be recorded

**Example:**

```go
// Create an adapter to capture logs
adapter := logger.GetStdLogAdapter(structuredLogger, logger.WARN)

// Use the adapter with any io.Writer interface
fileLogger := log.New(adapter, "FILE: ", log.Lshortfile)
```

#### `CreateStdLogProxy`

```go
func CreateStdLogProxy(logger *Logger, level LogLevel) *log.Logger
```

Creates a standard library logger that proxies to a ProjectMemory structured logger. This is useful when you need to pass a `*log.Logger` to a library but want the output to go through your structured logging system.

**Parameters:**

- `logger`: The ProjectMemory structured logger to use
- `level`: The log level at which logs should be recorded

**Example:**

```go
// Create a logger for a specific component with additional context
dbLogger := structuredLogger.WithField("component", "database")

// Create a proxy standard logger
stdLogger := logger.CreateStdLogProxy(dbLogger, logger.ERROR)

// Pass to code that requires a standard library logger
databaseLib.SetLogger(stdLogger)
```

## Technical Details

### How It Works

When you call `ReplaceStdLogger()`, the following happens:

1. A specialized adapter that implements Go's `io.Writer` interface is created
2. This adapter captures all standard log output and routes it through your structured logger
3. The standard library's global logger is reconfigured to use this adapter
4. All existing code using the standard logger automatically flows through your structured logger

The replacement affects **ALL** code in your process:

- Your own application code using `log.*` functions
- Third-party libraries and dependencies using the standard logger
- Any framework code relying on the standard logging package

### Implementation Notes

- The standard library's prefix and flags are reset when using `ReplaceStdLogger`
- ProjectMemory's logger adds its own timestamp, level, and context information
- Line numbers in logs will point to the actual log call in your code, not the adapter
- Log levels are mapped from standard library to ProjectMemory's level system

## Use Cases

### Gradual Migration

For large codebases with extensive use of the standard logger, this feature allows you to:

1. Add structured logging capabilities without changing existing log calls
2. Keep third-party libraries' logs consistent with your application logs
3. Implement a phased migration to fully structured logging

### Third-Party Libraries

When working with libraries that use the standard logger internally:

1. Pass a proxy logger configured for the appropriate log level
2. Get properly formatted structured logs from code you don't control
3. Attach additional context (like component name, request ID) to library logs

### Backward Compatibility

When maintaining backward compatibility is important:

1. Continue supporting existing code patterns
2. Enhance logging capabilities without breaking changes
3. Ensure all logs follow a consistent format and level structure

## Additional Resources

- [Example Implementation](../examples/stdlog-integration/main.go)
- [Integration README](../examples/stdlog-integration/README.md)
