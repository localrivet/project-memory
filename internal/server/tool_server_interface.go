// Package server provides the MCP server implementation for the Project-Memory service.
package server

// ContextToolServer defines the interface for the MCP server that handles
// context-related tool calls from MCP clients.
type ContextToolServer interface {
	// Initialize initializes the server with dependencies and configurations.
	Initialize() error

	// Start starts the MCP server on the specified transport.
	Start() error

	// Stop gracefully shuts down the MCP server.
	Stop() error
}
