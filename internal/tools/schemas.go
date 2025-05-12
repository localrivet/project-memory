// Package tools defines the interfaces and data structures
// for the ProjectMemory service.
package tools

const (
	// ToolSaveContext is the name of the save_context MCP tool
	ToolSaveContext = "save_context"

	// ToolRetrieveContext is the name of the retrieve_context MCP tool
	ToolRetrieveContext = "retrieve_context"

	// ToolDeleteContext is the name of the delete_context MCP tool
	ToolDeleteContext = "delete_context"

	// ToolClearAllContext is the name of the clear_all_context MCP tool
	ToolClearAllContext = "clear_all_context"

	// ToolReplaceContext is the name of the replace_context MCP tool
	ToolReplaceContext = "replace_context"

	// DefaultRetrieveLimit is the default number of results to return
	// when no limit is specified in a retrieve_context request
	DefaultRetrieveLimit = 5
)

// SaveContextRequest defines the input schema for save_context tool
type SaveContextRequest struct {
	// ContextText is the text to save in the context store
	ContextText string `json:"context_text"`
}

// SaveContextResponse defines the output schema for save_context tool
type SaveContextResponse struct {
	// Status indicates the result of the operation ("success" or "error")
	Status string `json:"status"`

	// ID is the unique identifier assigned to the saved context
	ID string `json:"id"`

	// Error contains an error message if Status is "error"
	Error string `json:"error,omitempty"`
}

// RetrieveContextRequest defines the input schema for retrieve_context tool
type RetrieveContextRequest struct {
	// Query is the text to search for in the context store
	Query string `json:"query"`

	// Limit is the maximum number of results to return
	// If not specified, DefaultRetrieveLimit will be used
	Limit int `json:"limit,omitempty"`
}

// RetrieveContextResponse defines the output schema for retrieve_context tool
type RetrieveContextResponse struct {
	// Status indicates the result of the operation ("success" or "error")
	Status string `json:"status"`

	// Results contains the matching context entries
	Results []string `json:"results"`

	// Error contains an error message if Status is "error"
	Error string `json:"error,omitempty"`
}

// DeleteContextRequest defines the input schema for delete_context tool
type DeleteContextRequest struct {
	// ID is the unique identifier of the context entry to delete
	ID string `json:"id"`
}

// DeleteContextResponse defines the output schema for delete_context tool
type DeleteContextResponse struct {
	// Status indicates the result of the operation ("success" or "error")
	Status string `json:"status"`

	// Error contains an error message if Status is "error"
	Error string `json:"error,omitempty"`
}

// ClearAllContextRequest defines the input schema for clear_all_context tool
type ClearAllContextRequest struct {
	// Confirmation is a required field to confirm the operation
	// Must be set to "confirm" to prevent accidental clearing
	Confirmation string `json:"confirmation"`
}

// ClearAllContextResponse defines the output schema for clear_all_context tool
type ClearAllContextResponse struct {
	// Status indicates the result of the operation ("success" or "error")
	Status string `json:"status"`

	// Error contains an error message if Status is "error"
	Error string `json:"error,omitempty"`
}

// ReplaceContextRequest defines the input schema for replace_context tool
type ReplaceContextRequest struct {
	// ID is the unique identifier of the context entry to replace
	ID string `json:"id"`

	// ContextText is the new text to replace the existing context
	ContextText string `json:"context_text"`
}

// ReplaceContextResponse defines the output schema for replace_context tool
type ReplaceContextResponse struct {
	// Status indicates the result of the operation ("success" or "error")
	Status string `json:"status"`

	// Error contains an error message if Status is "error"
	Error string `json:"error,omitempty"`
}
