package vector

import (
	"crypto/md5"
	"encoding/binary"
	"math"
)

// MockEmbedder is a simple implementation of the Embedder interface.
// It creates deterministic but simplistic embeddings for testing purposes.
type MockEmbedder struct {
	dimensions int
}

// NewMockEmbedder creates a new MockEmbedder with the specified dimensions.
func NewMockEmbedder(dimensions int) *MockEmbedder {
	if dimensions <= 0 {
		dimensions = 128 // Default dimension
	}
	return &MockEmbedder{
		dimensions: dimensions,
	}
}

// Initialize sets up the embedder with any required configuration.
func (e *MockEmbedder) Initialize() error {
	return nil // No initialization needed for the mock embedder
}

// CreateEmbedding generates a mock embedding for the given text.
// It uses a deterministic algorithm based on MD5 hashing to ensure
// that the same text always produces the same embedding.
func (e *MockEmbedder) CreateEmbedding(text string) ([]float32, error) {
	// Create an embedding of the specified dimensions
	embedding := make([]float32, e.dimensions)

	// Use MD5 hash of the text as a seed for the embedding
	hash := md5.Sum([]byte(text))

	// Fill the embedding array with values derived from the hash
	for i := 0; i < e.dimensions; i++ {
		// Use 4 bytes from the hash as a seed for each dimension
		// Wrap around the hash if needed
		hashIdx := (i * 4) % len(hash)
		seed := binary.LittleEndian.Uint32(append(hash[hashIdx:], hash[:4]...))

		// Generate a value between -1 and 1 based on the seed
		value := float32(seed%1000)/500.0 - 1.0
		embedding[i] = value
	}

	// Normalize the embedding
	e.normalizeEmbedding(embedding)

	return embedding, nil
}

// normalizeEmbedding normalizes the embedding to have unit length.
func (e *MockEmbedder) normalizeEmbedding(embedding []float32) {
	// Calculate the squared magnitude
	var sumSquares float32
	for _, val := range embedding {
		sumSquares += val * val
	}

	// Calculate the magnitude
	magnitude := float32(math.Sqrt(float64(sumSquares)))

	// Normalize each component
	for i := range embedding {
		embedding[i] /= magnitude
	}
}
