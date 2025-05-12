# Configuration Reference

Project-Memory uses a JSON configuration file to customize its behavior. This document provides a detailed reference for all available configuration options.

## Configuration File

The configuration file should be named `.projectmemoryconfig` and placed in the root directory where you run the application.

## Basic Configuration Structure

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

## Configuration Sections

### Models Section

The `models` section configures the AI model used for summarization:

| Option        | Type    | Description                                                   | Default      |
| ------------- | ------- | ------------------------------------------------------------- | ------------ |
| `provider`    | string  | The AI provider to use (mock, anthropic, openai, google, xai) | "mock"       |
| `modelId`     | string  | The specific model ID to use                                  | "mock-model" |
| `maxTokens`   | integer | Maximum number of tokens for model responses                  | 1024         |
| `temperature` | float   | Creativity level (0.0-1.0)                                    | 0.7          |

For example, to use Anthropic's Claude:

```json
"models": {
  "provider": "anthropic",
  "modelId": "claude-3-opus-20240229",
  "maxTokens": 4096,
  "temperature": 0.2
}
```

### Summarizer Section (Optional)

The `summarizer` section provides additional configuration for text summarization:

| Option              | Type    | Description                                 | Default                 |
| ------------------- | ------- | ------------------------------------------- | ----------------------- |
| `provider`          | string  | AI provider for summarization               | Same as models.provider |
| `modelId`           | string  | Model for summarization                     | Same as models.modelId  |
| `maxSummaryLength`  | integer | Maximum length of generated summaries       | 500                     |
| `preserveKeyTerms`  | boolean | Whether to preserve key terms in summaries  | true                    |
| `fallbackProviders` | array   | List of fallback providers if primary fails | []                      |

Example:

```json
"summarizer": {
  "provider": "anthropic",
  "modelId": "claude-3-opus-20240229",
  "maxSummaryLength": 500,
  "preserveKeyTerms": true,
  "fallbackProviders": [
    {
      "provider": "openai",
      "modelId": "gpt-4o"
    },
    {
      "provider": "google",
      "modelId": "gemini-1.5-pro"
    }
  ]
}
```

### Database Section

The `database` section configures the SQLite database:

| Option | Type   | Description                      | Default             |
| ------ | ------ | -------------------------------- | ------------------- |
| `path` | string | Path to the SQLite database file | ".projectmemory.db" |

### Logging Section

The `logging` section configures the logging system:

| Option   | Type   | Description                                 | Default |
| -------- | ------ | ------------------------------------------- | ------- |
| `level`  | string | Log level (DEBUG, INFO, WARN, ERROR, FATAL) | "INFO"  |
| `format` | string | Log format (TEXT, JSON)                     | "TEXT"  |

### API Keys Section (Optional)

The `apiKeys` section stores API keys for various providers. It's recommended to use environment variables instead for security reasons:

```json
"apiKeys": {
  "anthropic": "${ANTHROPIC_API_KEY}",
  "openai": "${OPENAI_API_KEY}",
  "google": "${GOOGLE_API_KEY}",
  "xai": "${XAI_API_KEY}"
}
```

### Cache Section (Optional)

The `cache` section configures caching behavior:

| Option     | Type    | Description                    | Default |
| ---------- | ------- | ------------------------------ | ------- |
| `enabled`  | boolean | Whether caching is enabled     | true    |
| `ttl`      | integer | Time-to-live in seconds        | 86400   |
| `capacity` | integer | Maximum number of cached items | 1000    |

### Retry Section (Optional)

The `retry` section configures request retry behavior:

| Option         | Type    | Description                      | Default |
| -------------- | ------- | -------------------------------- | ------- |
| `maxRetries`   | integer | Maximum number of retry attempts | 3       |
| `initialDelay` | integer | Initial delay in milliseconds    | 1000    |
| `maxDelay`     | integer | Maximum delay in milliseconds    | 10000   |

## Environment Variables

Some configuration options can be overridden using environment variables:

| Environment Variable | Description                |
| -------------------- | -------------------------- |
| `LOG_LEVEL`          | Override the logging level |
| `ANTHROPIC_API_KEY`  | API key for Anthropic      |
| `OPENAI_API_KEY`     | API key for OpenAI         |
| `GOOGLE_API_KEY`     | API key for Google AI      |
| `XAI_API_KEY`        | API key for XAI            |

## Configuration Best Practices

1. **Security**: Don't commit API keys in the config file; use environment variables
2. **Development**: Use the mock provider during development to avoid API costs
3. **Tuning**: Adjust maxSummaryLength based on your use case and token usage
4. **Logging**: Use INFO level in production; DEBUG for development
