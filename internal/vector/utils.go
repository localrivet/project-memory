package vector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// Float32SliceToBytes converts a slice of float32 to a byte slice.
func Float32SliceToBytes(floats []float32) ([]byte, error) {
	buf := new(bytes.Buffer)

	// First write the length of the slice
	err := binary.Write(buf, binary.LittleEndian, int32(len(floats)))
	if err != nil {
		return nil, fmt.Errorf("failed to write vector length: %w", err)
	}

	// Then write the float32 values
	err = binary.Write(buf, binary.LittleEndian, floats)
	if err != nil {
		return nil, fmt.Errorf("failed to write vector values: %w", err)
	}

	return buf.Bytes(), nil
}

// BytesToFloat32Slice converts a byte slice to a slice of float32.
func BytesToFloat32Slice(data []byte) ([]float32, error) {
	buf := bytes.NewReader(data)

	// First read the length of the slice
	var length int32
	err := binary.Read(buf, binary.LittleEndian, &length)
	if err != nil {
		return nil, fmt.Errorf("failed to read vector length: %w", err)
	}

	// Then read the float32 values
	floats := make([]float32, length)
	err = binary.Read(buf, binary.LittleEndian, floats)
	if err != nil {
		return nil, fmt.Errorf("failed to read vector values: %w", err)
	}

	return floats, nil
}

// CosineSimilarity calculates the cosine similarity between two vectors.
// The result is a value between -1 and 1, where 1 means the vectors are identical,
// 0 means they are orthogonal, and -1 means they are opposite.
func CosineSimilarity(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same dimension: %d != %d", len(a), len(b))
	}

	// Calculate dot product
	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	// Check for zero vectors
	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("one or both vectors have zero magnitude")
	}

	// Calculate cosine similarity
	similarity := float64(dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))))

	return similarity, nil
}
