package vector

import (
	"math"
	"reflect"
	"testing"
)

func TestFloat32SliceToBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []float32
	}{
		{
			name:  "empty slice",
			input: []float32{},
		},
		{
			name:  "single value",
			input: []float32{1.0},
		},
		{
			name:  "multiple values",
			input: []float32{1.0, 2.0, 3.0, 4.0, 5.0},
		},
		{
			name:  "negative values",
			input: []float32{-1.0, -2.0, -3.0, -4.0, -5.0},
		},
		{
			name:  "mixed values",
			input: []float32{-1.0, 0.0, 1.0, 3.14, -2.718},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Convert to bytes
			bytes, err := Float32SliceToBytes(test.input)
			if err != nil {
				t.Errorf("Float32SliceToBytes(%v) error: %v", test.input, err)
				return
			}

			// Convert back to float32 slice
			floats, err := BytesToFloat32Slice(bytes)
			if err != nil {
				t.Errorf("BytesToFloat32Slice(%v) error: %v", bytes, err)
				return
			}

			// Verify the result matches the input
			if !reflect.DeepEqual(test.input, floats) {
				t.Errorf("Expected %v, got %v", test.input, floats)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
		wantErr  bool
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{1.0, 2.0, 3.0},
			expected: 1.0,
			wantErr:  false,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{0.0, 1.0, 0.0},
			expected: 0.0,
			wantErr:  false,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{-1.0, -2.0, -3.0},
			expected: -1.0,
			wantErr:  false,
		},
		{
			name:     "different length vectors",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{1.0, 2.0},
			expected: 0.0,
			wantErr:  true,
		},
		{
			name:     "zero vector",
			a:        []float32{0.0, 0.0, 0.0},
			b:        []float32{1.0, 2.0, 3.0},
			expected: 0.0,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			similarity, err := CosineSimilarity(test.a, test.b)

			// Check error
			if (err != nil) != test.wantErr {
				t.Errorf("CosineSimilarity() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			// If we expect an error, don't check the similarity value
			if test.wantErr {
				return
			}

			// Check similarity value
			if math.Abs(similarity-test.expected) > 1e-6 {
				t.Errorf("CosineSimilarity() = %v, want %v", similarity, test.expected)
			}
		})
	}
}

func TestMockEmbedder(t *testing.T) {
	embedder := NewMockEmbedder(128)

	// Test initialization
	err := embedder.Initialize()
	if err != nil {
		t.Errorf("MockEmbedder.Initialize() error = %v", err)
		return
	}

	// Test embedding creation
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "short text",
			input: "Hello, world!",
		},
		{
			name:  "longer text",
			input: "This is a longer piece of text to test the embedding functionality.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create embedding
			embedding, err := embedder.CreateEmbedding(test.input)
			if err != nil {
				t.Errorf("MockEmbedder.CreateEmbedding(%q) error = %v", test.input, err)
				return
			}

			// Check dimensions
			if len(embedding) != 128 {
				t.Errorf("Expected embedding dimension 128, got %d", len(embedding))
			}

			// Check unit length (normalization)
			var sumSquares float32
			for _, val := range embedding {
				sumSquares += val * val
			}
			magnitude := float64(math.Sqrt(float64(sumSquares)))
			if math.Abs(magnitude-1.0) > 1e-6 {
				t.Errorf("Expected unit vector (magnitude 1.0), got %f", magnitude)
			}

			// Create embedding for the same input again and verify it's deterministic
			embedding2, err := embedder.CreateEmbedding(test.input)
			if err != nil {
				t.Errorf("MockEmbedder.CreateEmbedding(%q) 2nd call error = %v", test.input, err)
				return
			}

			if !reflect.DeepEqual(embedding, embedding2) {
				t.Errorf("Expected identical embeddings for the same input, but they differ")
			}
		})
	}
}
