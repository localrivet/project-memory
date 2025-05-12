// Package tools defines the MCP tool schemas and constants
// for the Project-Memory service.
package tools

const (
	// ToolSaveContext is the name of the save_context MCP tool
	ToolSaveContext = "save_context"

	// ToolRetrieveContext is the name of the retrieve_context MCP tool
	ToolRetrieveContext = "retrieve_context"

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
