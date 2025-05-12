# API Reference

Project-Memory provides MCP tools for saving and retrieving contextual information. This document details the available tools, their request/response formats, and usage examples.

## MCP Tools Overview

Project-Memory exposes two main MCP tools:

1. `save_context` - Saves a piece of text to the context store
2. `retrieve_context` - Retrieves relevant context based on a query

## Tool: save_context

The `save_context` tool stores a context snippet in the database, summarizing it and creating an embedding for future similarity searches.

### Request Format

```json
{
  "context_text": "The text to save in the context store"
}
```

#### Parameters

| Parameter      | Type   | Description                                   | Required |
| -------------- | ------ | --------------------------------------------- | -------- |
| `context_text` | string | The text content to save in the context store | Yes      |

### Response Format

```json
{
  "status": "success",
  "id": "generated-unique-id"
}
```

#### Response Fields

| Field    | Type   | Description                                         |
| -------- | ------ | --------------------------------------------------- |
| `status` | string | The result of the operation: "success" or "error"   |
| `id`     | string | The unique identifier assigned to the saved context |
| `error`  | string | Error message (only present if status is "error")   |

### Example

**Request:**

```json
{
  "context_text": "Project-Memory is a Go application that allows Large Language Models to maintain context across interactions by storing and retrieving relevant information using vector similarity search."
}
```

**Response:**

```json
{
  "status": "success",
  "id": "d8e8fca2dc0f896"
}
```

## Tool: retrieve_context

The `retrieve_context` tool searches for and returns the most similar context snippets to a given query.

### Request Format

```json
{
  "query": "The query to search for",
  "limit": 5
}
```

#### Parameters

| Parameter | Type    | Description                                      | Required |
| --------- | ------- | ------------------------------------------------ | -------- |
| `query`   | string  | The text to search for in the context store      | Yes      |
| `limit`   | integer | Maximum number of results to return (default: 5) | No       |

### Response Format

```json
{
  "status": "success",
  "results": ["Matching context entry 1", "Matching context entry 2", "..."]
}
```

#### Response Fields

| Field     | Type   | Description                                       |
| --------- | ------ | ------------------------------------------------- |
| `status`  | string | The result of the operation: "success" or "error" |
| `results` | array  | List of matching context entries                  |
| `error`   | string | Error message (only present if status is "error") |

### Example

**Request:**

```json
{
  "query": "How does Project-Memory store context?",
  "limit": 2
}
```

**Response:**

```json
{
  "status": "success",
  "results": [
    "Project-Memory is a Go application that allows Large Language Models to maintain context across interactions by storing and retrieving relevant information using vector similarity search.",
    "Project-Memory stores context in a SQLite database with embeddings to enable semantic search."
  ]
}
```

## Error Handling

All tools return a standardized error format when an error occurs:

```json
{
  "status": "error",
  "error": "Detailed error message"
}
```

Common error scenarios include:

- Invalid input parameters
- Database connection issues
- Embedding creation failures
- Rate limiting from AI providers

## Using the API with gomcp

Here's an example of how to call these tools using the gomcp library:

```go
package main

import (
	"fmt"
	"github.com/localrivet/gomcp/client"
)

func main() {
	// Create MCP client
	c := client.NewClient()

	// Save context
	saveReq := map[string]interface{}{
		"context_text": "This is important information to remember.",
	}

	var saveResp map[string]interface{}
	if err := c.CallTool("save_context", saveReq, &saveResp); err != nil {
		fmt.Printf("Error saving context: %v\n", err)
		return
	}

	fmt.Printf("Context saved with ID: %s\n", saveResp["id"])

	// Retrieve context
	retrieveReq := map[string]interface{}{
		"query": "What information do we have?",
		"limit": 3,
	}

	var retrieveResp map[string]interface{}
	if err := c.CallTool("retrieve_context", retrieveReq, &retrieveResp); err != nil {
		fmt.Printf("Error retrieving context: %v\n", err)
		return
	}

	fmt.Println("Retrieved context:")
	for i, result := range retrieveResp["results"].([]interface{}) {
		fmt.Printf("%d. %s\n", i+1, result)
	}
}
```

## Using as a Library

Project-Memory can also be embedded as a library in your Go application. See the [Development Guide](development.md) for details.
