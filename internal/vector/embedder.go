// Package vector provides interfaces and utilities for vector operations
// and text embedding within the Project-Memory service.
package vector

const (
	// DefaultEmbeddingDimensions defines the standard size of embedding vectors.
	// 1536 is a common size for modern embedding models.
	DefaultEmbeddingDimensions = 1536

	// DefaultBatchSize defines how many embeddings can be processed in a single batch.
	DefaultBatchSize = 8
)

// Embedder defines the interface for creating vector embeddings from text.
type Embedder interface {
	// CreateEmbedding converts text into a vector representation.
	CreateEmbedding(text string) ([]float32, error)

	// Initialize sets up the embedder with any required configuration.
	Initialize() error
}
