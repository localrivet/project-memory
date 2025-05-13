package util

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// GenerateHash creates a hash from the summary and a timestamp
func GenerateHash(summary string, timestamp int64) string {
	hasher := sha256.New()
	hasher.Write([]byte(summary))
	hasher.Write([]byte(time.Unix(0, timestamp).String()))
	return hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars of the hash
}
