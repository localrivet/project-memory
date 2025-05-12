// Package contextstore provides the storage components for
// the context data used by the ProjectMemory service.
package contextstore

import (
	"time"
)

// ContextStore defines the interface for storing and retrieving context data.
type ContextStore interface {
	// Initialize initializes the store with configuration options.
	Initialize(dbPath string) error

	// Close closes the store and releases any resources.
	Close() error

	// Store stores the context data in the database.
	Store(id string, summaryText string, embedding []byte, timestamp time.Time) error

	// Search searches for context entries similar to the given embedding.
	Search(queryEmbedding []float32, limit int) ([]string, error)

	// DeleteContext deletes a specific context entry from the store by ID.
	DeleteContext(id string) error

	// ClearAllContext removes all context entries from the store.
	ClearAllContext() error

	// ReplaceContext replaces a context entry with updated information.
	// Note: The current Store method performs replacement when an ID already exists,
	// but this method makes the intent clearer.
	ReplaceContext(id string, summaryText string, embedding []byte, timestamp time.Time) error
}
