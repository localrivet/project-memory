package contextstore

import (
	"fmt"
	"sort"
	"time"

	"crawshaw.io/sqlite"
	"github.com/localrivet/project-memory/internal/vector"
)

// SQLiteContextStore is an implementation of ContextStore that uses SQLite.
type SQLiteContextStore struct {
	conn   *sqlite.Conn
	dbPath string
}

// NewSQLiteContextStore creates a new SQLiteContextStore instance.
func NewSQLiteContextStore() *SQLiteContextStore {
	return &SQLiteContextStore{}
}

// Initialize initializes the store with the given database path.
func (s *SQLiteContextStore) Initialize(dbPath string) error {
	s.dbPath = dbPath

	// Open the SQLite database
	conn, err := sqlite.OpenConn(dbPath, sqlite.SQLITE_OPEN_CREATE|sqlite.SQLITE_OPEN_READWRITE)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}
	s.conn = conn

	// Create the table if it doesn't exist
	err = s.createTable()
	if err != nil {
		// Close the connection on error
		s.conn.Close()
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// createTable creates the context_memory table if it doesn't exist.
func (s *SQLiteContextStore) createTable() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS context_memory (
		id TEXT PRIMARY KEY,
		summary_text TEXT NOT NULL,
		embedding BLOB NOT NULL,
		timestamp INTEGER NOT NULL
	);`

	stmt, err := s.conn.Prepare(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare create table statement: %w", err)
	}
	defer stmt.Reset()

	_, err = stmt.Step()
	if err != nil {
		return fmt.Errorf("failed to execute create table statement: %w", err)
	}

	return nil
}

// Close closes the store and releases any resources.
func (s *SQLiteContextStore) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// Store stores the context data in the database.
func (s *SQLiteContextStore) Store(id string, summaryText string, embedding []byte, timestamp time.Time) error {
	// Insert or replace the context entry
	insertSQL := `
	INSERT OR REPLACE INTO context_memory (id, summary_text, embedding, timestamp)
	VALUES (?, ?, ?, ?);`

	stmt, err := s.conn.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Reset()

	// Bind parameters - indices in sqlite are 1-based
	stmt.BindText(1, id)
	stmt.BindText(2, summaryText)
	stmt.BindBytes(3, embedding)
	stmt.BindInt64(4, timestamp.Unix())

	// Execute the statement
	_, err = stmt.Step()
	if err != nil {
		return fmt.Errorf("failed to insert context entry: %w", err)
	}

	return nil
}

// Search searches for context entries similar to the given embedding.
func (s *SQLiteContextStore) Search(queryEmbedding []float32, limit int) ([]string, error) {
	// First, convert query embedding to bytes for debugging purposes
	// (won't be used directly for search as we'll do similarity calculations in Go)
	_, err := vector.Float32SliceToBytes(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("failed to convert query embedding to bytes: %w", err)
	}

	// Retrieve all entries from the database
	selectSQL := `
	SELECT id, summary_text, embedding FROM context_memory
	ORDER BY timestamp DESC;`

	stmt, err := s.conn.Prepare(selectSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Reset()

	// Maps to store results for sorting
	type Result struct {
		SummaryText string
		Similarity  float64
	}
	var results []Result

	// Execute the query and process results
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("failed to execute select statement: %w", err)
		}
		if !hasRow {
			break // No more rows
		}

		// Get values from the current row
		// Column indices are 0-based
		id := stmt.ColumnText(0)
		summaryText := stmt.ColumnText(1)

		// For binary data, we need to create a buffer and use ColumnBytes to fill it
		embeddingBytesLen := stmt.ColumnLen(2)
		embeddingBytes := make([]byte, embeddingBytesLen)
		stmt.ColumnBytes(2, embeddingBytes)

		// Convert embedding bytes to float32 slice
		storedEmbedding, err := vector.BytesToFloat32Slice(embeddingBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to convert embedding bytes for entry %s: %w", id, err)
		}

		// Calculate cosine similarity
		similarity, err := vector.CosineSimilarity(queryEmbedding, storedEmbedding)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate similarity for entry %s: %w", id, err)
		}

		// Add to results
		results = append(results, Result{
			SummaryText: summaryText,
			Similarity:  similarity,
		})
	}

	// Sort results by similarity (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// If limit is greater than available results, adjust it
	if limit > len(results) {
		limit = len(results)
	}

	// Extract the top summaries
	topSummaries := make([]string, limit)
	for i := 0; i < limit; i++ {
		if i < len(results) {
			topSummaries[i] = results[i].SummaryText
		}
	}

	return topSummaries, nil
}
