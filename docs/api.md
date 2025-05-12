# API Reference

ProjectMemory provides MCP tools for saving, retrieving, and managing contextual information. This document details the available tools, their request/response formats, and usage examples.

## MCP Tools Overview

ProjectMemory exposes five main MCP tools:

1. `save_context` - Saves a piece of text to the context store
2. `retrieve_context` - Retrieves relevant context based on a query
3. `delete_context` - Deletes a specific context entry by ID
4. `clear_all_context` - Removes all context entries from the store
5. `replace_context` - Replaces an existing context entry with new content

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
  "context_text": "ProjectMemory is a Go application that allows Large Language Models to maintain context across interactions by storing and retrieving relevant information using vector similarity search."
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
  "query": "How does ProjectMemory store context?",
  "limit": 2
}
```

**Response:**

```json
{
  "status": "success",
  "results": [
    "ProjectMemory is a Go application that allows Large Language Models to maintain context across interactions by storing and retrieving relevant information using vector similarity search.",
    "ProjectMemory stores context in a SQLite database with embeddings to enable semantic search."
  ]
}
```

## Tool: delete_context

The `delete_context` tool removes a specific context entry from the store using its unique ID.

### Request Format

```json
{
  "id": "context-entry-id-to-delete"
}
```

#### Parameters

| Parameter | Type   | Description                                    | Required |
| --------- | ------ | ---------------------------------------------- | -------- |
| `id`      | string | The unique identifier of the context to delete | Yes      |

### Response Format

```json
{
  "status": "success"
}
```

#### Response Fields

| Field    | Type   | Description                                       |
| -------- | ------ | ------------------------------------------------- |
| `status` | string | The result of the operation: "success" or "error" |
| `error`  | string | Error message (only present if status is "error") |

### Example

**Request:**

```json
{
  "id": "d8e8fca2dc0f896"
}
```

**Response:**

```json
{
  "status": "success"
}
```

## Tool: clear_all_context

The `clear_all_context` tool removes all context entries from the store. This is a destructive operation, so it requires explicit confirmation.

### Request Format

```json
{
  "confirmation": "confirm"
}
```

#### Parameters

| Parameter      | Type   | Description                                                     | Required |
| -------------- | ------ | --------------------------------------------------------------- | -------- |
| `confirmation` | string | Must be exactly "confirm" to proceed with clearing all contexts | Yes      |

### Response Format

```json
{
  "status": "success"
}
```

#### Response Fields

| Field    | Type   | Description                                       |
| -------- | ------ | ------------------------------------------------- |
| `status` | string | The result of the operation: "success" or "error" |
| `error`  | string | Error message (only present if status is "error") |

### Example

**Request:**

```json
{
  "confirmation": "confirm"
}
```

**Response:**

```json
{
  "status": "success"
}
```

## Tool: replace_context

The `replace_context` tool replaces an existing context entry with new content, updating its summary and embedding.

### Request Format

```json
{
  "id": "context-entry-id-to-replace",
  "context_text": "The new text to replace the existing context"
}
```

#### Parameters

| Parameter      | Type   | Description                                          | Required |
| -------------- | ------ | ---------------------------------------------------- | -------- |
| `id`           | string | The unique identifier of the context to replace      | Yes      |
| `context_text` | string | The new text content to replace the existing context | Yes      |

### Response Format

```json
{
  "status": "success"
}
```

#### Response Fields

| Field    | Type   | Description                                       |
| -------- | ------ | ------------------------------------------------- |
| `status` | string | The result of the operation: "success" or "error" |
| `error`  | string | Error message (only present if status is "error") |

### Example

**Request:**

```json
{
  "id": "d8e8fca2dc0f896",
  "context_text": "Updated description: ProjectMemory is a Go MCP server that provides persistent context storage using SQLite and vector embeddings."
}
```

**Response:**

```json
{
  "status": "success"
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
- Context entry not found (for delete/replace operations)
- Missing or invalid confirmation for clear all operation

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

ProjectMemory can also be embedded as a library in your Go application. See the [Library Usage Guide](library_usage.md) and [Embedding Guide](embedding_guide.md) for detailed information.
