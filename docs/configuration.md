# Configuration Reference

Project-Memory uses a JSON configuration file to customize its behavior. The configuration can be loaded from a file or set programmatically. You can also override certain settings using environment variables.

## Configuration File

The configuration file should be named `.projectmemoryconfig` and placed in the root directory where you run the application.

## Basic Configuration Structure

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

## Configuration Sections

### Store Section

The `store` section configures the data storage:

| Option        | Type   | Description                      | Environment Variable | Default             | Validation |
| ------------- | ------ | -------------------------------- | -------------------- | ------------------- | ---------- |
| `sqlite_path` | string | Path to the SQLite database file | `SQLITE_PATH`        | ".projectmemory.db" | `required` |

### Summarizer Section

The `summarizer` section configures the text summarization:

| Option     | Type   | Description                            | Environment Variable  | Default |
| ---------- | ------ | -------------------------------------- | --------------------- | ------- |
| `provider` | string | The summarization provider to use      | `SUMMARIZER_PROVIDER` | "basic" |
| `api_key`  | string | API key for the summarization provider | `SUMMARIZER_API_KEY`  | ""      |

### Embedder Section

The `embedder` section configures the embedding generation:

| Option       | Type    | Description                        | Environment Variable  | Default | Validation |
| ------------ | ------- | ---------------------------------- | --------------------- | ------- | ---------- |
| `provider`   | string  | The embedding provider to use      | `EMBEDDER_PROVIDER`   | "mock"  |            |
| `dimensions` | integer | Dimensions for the embeddings      | `EMBEDDER_DIMENSIONS` | 768     | `min:1`    |
| `api_key`    | string  | API key for the embedding provider | `EMBEDDER_API_KEY`    | ""      |            |

### Logging Section

The `logging` section configures the logging system:

| Option   | Type   | Description                          | Environment Variable | Default | Validation |
| -------- | ------ | ------------------------------------ | -------------------- | ------- | ---------- |
| `level`  | string | Log level (debug, info, warn, error) | `LOG_LEVEL`          | "info"  | `required` |
| `format` | string | Log format (text, json)              | `LOG_FORMAT`         | "text"  |            |

## Environment Variables

Configuration options can be overridden using environment variables. The environment variables take precedence over values specified in the configuration file. The naming convention for environment variables is to use uppercase with underscores.

ProjectMemory uses the `github.com/localrivet/configurator` library which supports a flexible environment variable resolution system. Environment variables are checked using the following formats:

1. `PROJECTMEMORY_SECTION_OPTION` (e.g., `PROJECTMEMORY_STORE_SQLITE_PATH`)
2. Direct mapping via tag (e.g., `SQLITE_PATH` as specified in the `env:` tag)

### Core Environment Variables

| Environment Variable                | Alt Environment Variable | Configuration Path  | Description                        |
| ----------------------------------- | ------------------------ | ------------------- | ---------------------------------- |
| `PROJECTMEMORY_STORE_SQLITE_PATH`   | `SQLITE_PATH`            | store.sqlite_path   | Path to the SQLite database        |
| `PROJECTMEMORY_SUMMARIZER_PROVIDER` | `SUMMARIZER_PROVIDER`    | summarizer.provider | The summarization provider         |
| `PROJECTMEMORY_SUMMARIZER_API_KEY`  | `SUMMARIZER_API_KEY`     | summarizer.api_key  | API key for summarization provider |
| `PROJECTMEMORY_EMBEDDER_PROVIDER`   | `EMBEDDER_PROVIDER`      | embedder.provider   | The embedding provider             |
| `PROJECTMEMORY_EMBEDDER_DIMENSIONS` | `EMBEDDER_DIMENSIONS`    | embedder.dimensions | Dimensions for embeddings          |
| `PROJECTMEMORY_EMBEDDER_API_KEY`    | `EMBEDDER_API_KEY`       | embedder.api_key    | API key for embedding provider     |
| `PROJECTMEMORY_LOGGING_LEVEL`       | `LOG_LEVEL`              | logging.level       | Log level                          |
| `PROJECTMEMORY_LOGGING_FORMAT`      | `LOG_FORMAT`             | logging.format      | Log format                         |

## Using the Configuration Package

ProjectMemory includes a configuration package that makes it easy to load, validate, and save configuration. This package is based on the `github.com/localrivet/configurator` library and provides additional functionality like environment variable support and validation.

### Loading Configuration

```go
import "github.com/localrivet/projectmemory"

// Load from default path
cfg, err := projectmemory.DefaultConfig()
if err != nil {
    // Handle error
}

// Load from custom path
cfg, err := projectmemory.LoadConfigWithPath("/path/to/config")
if err != nil {
    // Handle error
}
```

### Getting Default Configuration

```go
// Create a new configuration with default values
cfg := projectmemory.DefaultConfig()
```

### Saving Configuration

```go
// Save to the path it was loaded from
err := cfg.Save()
if err != nil {
    // Handle error
}

// Save to a specific path
err := cfg.SaveToFile("/path/to/config")
if err != nil {
    // Handle error
}
```

## Validation

The configuration system includes validation to ensure required fields are present and values are within expected ranges:

- `required`: Field must have a non-empty value
- `min:X`: Numeric field must be at least X (e.g., `min:1` for dimensions)

If validation fails, the configuration loading process will return an error with details about which fields failed validation.

## Configuration Best Practices

1. **Security**: Don't commit API keys in the config file; use environment variables instead
2. **Development**: Use the mock provider during development to avoid API costs
3. **Logging**: Use "info" level in production, "debug" for development
4. **Configuration Management**: Create a configuration file with your application's defaults and allow users to override specific options as needed
