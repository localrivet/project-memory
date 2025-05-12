package projectmemory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/localrivet/projectmemory/internal/contextstore"
	"github.com/localrivet/projectmemory/internal/logger"
	"github.com/localrivet/projectmemory/internal/server"
	"github.com/localrivet/projectmemory/internal/summarizer"
	"github.com/localrivet/projectmemory/internal/tools"
	"github.com/localrivet/projectmemory/internal/vector"
)

// Config represents the configuration structure for the Project-Memory service.
type Config struct {
	Models struct {
		Provider    string  `json:"provider"`
		ModelID     string  `json:"modelId"`
		MaxTokens   int     `json:"maxTokens"`
		Temperature float32 `json:"temperature"`
	} `json:"models"`
	Database struct {
		Path string `json:"path"`
	} `json:"database"`
	Logging struct {
		Level  string `json:"level"`
		Format string `json:"format"`
	} `json:"logging"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	config := &Config{}
	config.Models.Provider = "mock"
	config.Models.ModelID = "mock-model"
	config.Models.MaxTokens = 1024
	config.Models.Temperature = 0.7
	config.Database.Path = ".projectmemory.db"
	config.Logging.Level = "INFO"
	config.Logging.Format = "TEXT"
	return config
}

// Server represents a Project-Memory server instance
type Server struct {
	toolServer server.ContextToolServer
	store      contextstore.ContextStore
	summarizer summarizer.Summarizer
	embedder   vector.Embedder
	logger     *logger.Logger
}

// NewServer creates a new Project-Memory server with the provided configuration
func NewServer(config *Config) (*Server, error) {
	// Initialize logging
	loggerConfig := logger.DefaultConfig()
	if config.Logging.Level != "" {
		loggerConfig.Level = logger.ParseLevel(config.Logging.Level)
	}
	if config.Logging.Format == "json" {
		loggerConfig.Format = logger.JSON
	}

	appLogger := logger.New(loggerConfig)
	logger.SetDefaultLogger(appLogger)

	// Initialize context store
	store := contextstore.NewSQLiteContextStore()
	if err := store.Initialize(config.Database.Path); err != nil {
		return nil, logger.DatabaseError(err, "Failed to initialize SQLite context store")
	}

	// Initialize summarizer
	summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	if err := summ.Initialize(); err != nil {
		return nil, logger.ConfigError(err, "Failed to initialize summarizer")
	}

	// Initialize embedder
	emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
	if err := emb.Initialize(); err != nil {
		return nil, logger.ConfigError(err, "Failed to initialize embedder")
	}

	// Initialize server
	srv := server.NewContextToolServer(store, summ, emb)
	if err := srv.Initialize(); err != nil {
		return nil, logger.ConfigError(err, "Failed to initialize MCP server")
	}

	return &Server{
		toolServer: srv,
		store:      store,
		summarizer: summ,
		embedder:   emb,
		logger:     appLogger,
	}, nil
}

// Start starts the Project-Memory server
func (s *Server) Start() error {
	return s.toolServer.Start()
}

// Stop gracefully stops the Project-Memory server
func (s *Server) Stop() error {
	if err := s.toolServer.Stop(); err != nil {
		return err
	}
	return s.store.Close()
}

// LoadConfig loads configuration from the specified file path
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, logger.ConfigError(err, "failed to read config file")
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, logger.ConfigError(err, "failed to parse config file")
	}

	return &config, nil
}

// SaveContext saves a context text to the memory store
func (s *Server) SaveContext(contextText string) (string, error) {
	// Generate summary
	summary, err := s.summarizer.Summarize(contextText)
	if err != nil {
		return "", err
	}

	// Create embedding
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		return "", err
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		return "", err
	}

	// Generate ID
	id := GenerateHash(summary, time.Now().UnixNano())

	// Store in context store
	err = s.store.Store(id, summary, embeddingBytes, time.Now())
	if err != nil {
		return "", err
	}

	return id, nil
}

// RetrieveContext retrieves relevant context based on a query
func (s *Server) RetrieveContext(query string, limit int) ([]string, error) {
	// Set default limit if not specified
	if limit <= 0 {
		limit = tools.DefaultRetrieveLimit
	}

	// Create embedding for query
	queryEmbedding, err := s.embedder.CreateEmbedding(query)
	if err != nil {
		return nil, err
	}

	// Search context store
	results, err := s.store.Search(queryEmbedding, limit)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetLogger returns the server's logger
func (s *Server) GetLogger() *logger.Logger {
	return s.logger
}

// GetStore returns the server's context store
func (s *Server) GetStore() contextstore.ContextStore {
	return s.store
}

// GetSummarizer returns the server's summarizer
func (s *Server) GetSummarizer() summarizer.Summarizer {
	return s.summarizer
}

// GetEmbedder returns the server's embedder
func (s *Server) GetEmbedder() vector.Embedder {
	return s.embedder
}

// CreateComponents creates and initializes the core components without starting the server.
// This is useful when you want to use Project-Memory components in your own MCP server.
func CreateComponents(config *Config) (contextstore.ContextStore, summarizer.Summarizer, vector.Embedder, error) {
	// Initialize context store
	store := contextstore.NewSQLiteContextStore()
	if err := store.Initialize(config.Database.Path); err != nil {
		return nil, nil, nil, logger.DatabaseError(err, "Failed to initialize SQLite context store")
	}

	// Initialize summarizer
	summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	if err := summ.Initialize(); err != nil {
		return nil, nil, nil, logger.ConfigError(err, "Failed to initialize summarizer")
	}

	// Initialize embedder
	emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
	if err := emb.Initialize(); err != nil {
		return nil, nil, nil, logger.ConfigError(err, "Failed to initialize embedder")
	}

	return store, summ, emb, nil
}

// GenerateHash creates a SHA-256 hash from content and a timestamp
func GenerateHash(content string, timestamp int64) string {
	// Combine content and timestamp
	data := fmt.Sprintf("%s-%d", content, timestamp)

	// Create hash
	hash := sha256.Sum256([]byte(data))

	// Convert to hex string and return first 16 characters
	return hex.EncodeToString(hash[:])[:16]
}
