// Package contextstore provides storage interfaces and implementations for
// the context data used by the Project-Memory service.
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
}
