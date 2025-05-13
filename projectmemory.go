package projectmemory

import (
	"encoding/json"
	"os"
	"time"

	"github.com/localrivet/gomcp/logx"
	"github.com/localrivet/projectmemory/internal/config"
	"github.com/localrivet/projectmemory/internal/contextstore"
	"github.com/localrivet/projectmemory/internal/errortypes"
	"github.com/localrivet/projectmemory/internal/server"
	"github.com/localrivet/projectmemory/internal/summarizer"
	"github.com/localrivet/projectmemory/internal/util"
	"github.com/localrivet/projectmemory/internal/vector"
)

// Config represents the configuration for the ProjectMemory service.
type Config = config.Config

// Server represents the ProjectMemory service.
type Server struct {
	config     *config.Config
	store      contextstore.ContextStore
	summarizer summarizer.Summarizer
	embedder   vector.Embedder
	toolServer server.ContextToolServer
	logger     logx.Logger
}

// NewServer creates a new ProjectMemory Server with the given configuration path and logger.
func NewServer(configPath string, logger logx.Logger) (*Server, error) {
	// Use provided logger or create a default one
	if logger == nil {
		logger = logx.NewLogger("info")
	}

	// Load configuration
	logger.Debug("Loading configuration from %s", configPath)
	cfg, err := config.LoadConfigWithPath(configPath)
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		return nil, errortypes.ConfigError(err, "Failed to load configuration")
	}

	// Initialize SQLite context store
	logger.Debug("Initializing SQLite context store at %s", cfg.Store.SQLitePath)
	store := contextstore.NewSQLiteContextStore()
	err = store.Initialize(cfg.Store.SQLitePath)
	if err != nil {
		logger.Error("Failed to initialize SQLite context store: %v", err)
		return nil, errortypes.DatabaseError(err, "Failed to initialize SQLite context store")
	}

	// Initialize summarizer
	logger.Debug("Initializing summarizer with provider: %s", cfg.Summarizer.Provider)
	var sum summarizer.Summarizer
	switch cfg.Summarizer.Provider {
	case "basic", "":
		sum = summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	default:
		logger.Warn("Unknown summarizer provider: %s, using basic summarizer", cfg.Summarizer.Provider)
		sum = summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	}

	if err := sum.Initialize(); err != nil {
		logger.Error("Failed to initialize summarizer: %v", err)
		return nil, errortypes.ConfigError(err, "Failed to initialize summarizer")
	}

	// Initialize embedder
	logger.Debug("Initializing embedder with provider: %s, dimensions: %d", cfg.Embedder.Provider, cfg.Embedder.Dimensions)
	var emb vector.Embedder
	dimensions := cfg.Embedder.Dimensions
	if dimensions <= 0 {
		dimensions = vector.DefaultEmbeddingDimensions
	}

	switch cfg.Embedder.Provider {
	case "mock", "":
		emb = vector.NewMockEmbedder(dimensions)
	default:
		logger.Warn("Unknown embedder provider: %s, using mock embedder", cfg.Embedder.Provider)
		emb = vector.NewMockEmbedder(dimensions)
	}

	if err := emb.Initialize(); err != nil {
		logger.Error("Failed to initialize embedder: %v", err)
		return nil, errortypes.ConfigError(err, "Failed to initialize embedder")
	}

	// Create the MCP server
	logger.Debug("Initializing context tool server")
	mcpServer := server.NewContextToolServer(store, sum, emb)
	// Pass the logger to the context tool server
	mcpServer.WithLogger(logger)

	err = mcpServer.Initialize()
	if err != nil {
		logger.Error("Failed to initialize MCP server: %v", err)
		return nil, errortypes.ConfigError(err, "Failed to initialize MCP server")
	}

	logger.Info("ProjectMemory server successfully initialized")
	return &Server{
		config:     cfg,
		store:      store,
		summarizer: sum,
		embedder:   emb,
		toolServer: mcpServer,
		logger:     logger,
	}, nil
}

// DefaultConfig returns the default configuration for the ProjectMemory service.
func DefaultConfig() *Config {
	config := &Config{}
	config.Store.SQLitePath = ".projectmemory.db"
	config.Summarizer.Provider = "basic"
	config.Embedder.Provider = "mock"
	config.Embedder.Dimensions = vector.DefaultEmbeddingDimensions
	config.Logging.Level = "info"
	config.Logging.Format = "text"
	return config
}

// SaveConfig saves the configuration to a file and returns the JSON content.
func SaveConfig(config *Config, path string) ([]byte, error) {
	// Pretty-print the JSON for better readability
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, errortypes.ConfigError(err, "failed to marshal configuration")
	}

	return content, nil
}

// loadConfig loads the configuration from the given path.
func loadConfig(configPath string) (*Config, error) {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errortypes.ConfigError(err, "failed to read config file")
	}

	// Parse the config file
	config := &Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, errortypes.ConfigError(err, "failed to parse config file")
	}

	return config, nil
}

// Start starts the ProjectMemory service.
func (s *Server) Start() error {
	s.logger.Info("Starting ProjectMemory service")
	return s.toolServer.Start()
}

// Stop stops the ProjectMemory service.
func (s *Server) Stop() error {
	s.logger.Info("Stopping ProjectMemory service")
	err := s.toolServer.Stop()
	if err != nil {
		return err
	}

	// Close the store
	s.logger.Info("Closing store")
	err = s.store.Close()
	if err != nil {
		s.logger.Error("Failed to close store: %v", err)
		return err
	}

	s.logger.Info("ProjectMemory service stopped")
	return nil
}

// SaveContext saves the given text to the context store.
func (s *Server) SaveContext(text string) (string, error) {
	// Generate summary
	s.logger.Debug("Generating summary of text (length: %d)", len(text))
	summary, err := s.summarizer.Summarize(text)
	if err != nil {
		s.logger.Error("Failed to summarize text: %v", err)
		return "", err
	}

	// Create embedding
	s.logger.Debug("Creating embedding for summary")
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		s.logger.Error("Failed to create embedding: %v", err)
		return "", err
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		s.logger.Error("Failed to convert embedding to bytes: %v", err)
		return "", err
	}

	// Generate ID (simple hash of content + timestamp)
	timestamp := time.Now()
	id := GenerateHash(summary, timestamp.UnixNano())

	// Store in context store
	s.logger.Debug("Storing context with ID: %s", id)
	err = s.store.Store(id, summary, embeddingBytes, timestamp)
	if err != nil {
		s.logger.Error("Failed to store context: %v", err)
		return "", err
	}

	s.logger.Info("Successfully saved context with ID: %s", id)
	return id, nil
}

// RetrieveContext retrieves context entries similar to the given query.
func (s *Server) RetrieveContext(query string, limit int) ([]string, error) {
	// Create embedding for query
	s.logger.Debug("Creating embedding for query: %s", query)
	queryEmbedding, err := s.embedder.CreateEmbedding(query)
	if err != nil {
		s.logger.Error("Failed to create embedding for query: %v", err)
		return nil, err
	}

	// Search context store
	s.logger.Debug("Searching for similar context entries (limit: %d)", limit)
	results, err := s.store.Search(queryEmbedding, limit)
	if err != nil {
		s.logger.Error("Failed to search context store: %v", err)
		return nil, err
	}

	s.logger.Info("Retrieved %d context entries", len(results))
	return results, nil
}

// GetLogger returns the logger instance used by the server.
func (s *Server) GetLogger() logx.Logger {
	return s.logger
}

// GetStore returns the context store instance used by the server.
func (s *Server) GetStore() contextstore.ContextStore {
	return s.store
}

// GetSummarizer returns the summarizer instance used by the server.
func (s *Server) GetSummarizer() summarizer.Summarizer {
	return s.summarizer
}

// GetEmbedder returns the embedder instance used by the server.
func (s *Server) GetEmbedder() vector.Embedder {
	return s.embedder
}

// CreateComponents creates and initializes the components of the ProjectMemory service
// without creating a server instance. This is useful for components that need
// direct access to the store, summarizer, and embedder.
func CreateComponents(cfg *config.Config, logger logx.Logger) (contextstore.ContextStore, summarizer.Summarizer, vector.Embedder, error) {
	// Use provided logger or create a default one
	if logger == nil {
		logger = logx.NewLogger("info")
	}

	// Initialize SQLite context store
	logger.Debug("Initializing SQLite context store at %s", cfg.Store.SQLitePath)
	store := contextstore.NewSQLiteContextStore()
	err := store.Initialize(cfg.Store.SQLitePath)
	if err != nil {
		logger.Error("Failed to initialize SQLite context store: %v", err)
		return nil, nil, nil, errortypes.DatabaseError(err, "Failed to initialize SQLite context store")
	}

	// Initialize summarizer
	logger.Debug("Initializing summarizer with provider: %s", cfg.Summarizer.Provider)
	var sum summarizer.Summarizer
	switch cfg.Summarizer.Provider {
	case "basic", "":
		sum = summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	default:
		logger.Warn("Unknown summarizer provider: %s, using basic summarizer", cfg.Summarizer.Provider)
		sum = summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	}

	if err := sum.Initialize(); err != nil {
		logger.Error("Failed to initialize summarizer: %v", err)
		return nil, nil, nil, errortypes.ConfigError(err, "Failed to initialize summarizer")
	}

	// Initialize embedder
	logger.Debug("Initializing embedder with provider: %s, dimensions: %d", cfg.Embedder.Provider, cfg.Embedder.Dimensions)
	var emb vector.Embedder
	dimensions := cfg.Embedder.Dimensions
	if dimensions <= 0 {
		dimensions = vector.DefaultEmbeddingDimensions
	}

	switch cfg.Embedder.Provider {
	case "mock", "":
		emb = vector.NewMockEmbedder(dimensions)
	default:
		logger.Warn("Unknown embedder provider: %s, using mock embedder", cfg.Embedder.Provider)
		emb = vector.NewMockEmbedder(dimensions)
	}

	if err := emb.Initialize(); err != nil {
		logger.Error("Failed to initialize embedder: %v", err)
		return nil, nil, nil, errortypes.ConfigError(err, "Failed to initialize embedder")
	}

	logger.Info("Components successfully initialized")
	return store, sum, emb, nil
}

// WithLogger sets a custom logger for the server.
// This should be called immediately after creating the server with NewServer
// and before calling any other methods if you need to replace the logger used during initialization.
func (s *Server) WithLogger(customLogger logx.Logger) *Server {
	if customLogger == nil {
		return s
	}

	s.logger = customLogger

	// If the toolServer has been initialized, update its logger
	if mcp, ok := s.toolServer.(*server.MCPContextToolServer); ok && mcp != nil {
		mcp.WithLogger(customLogger)
	}

	return s
}

// GenerateHash creates a hash from the summary and a timestamp
// This is a convenience wrapper around the internal util.GenerateHash function
func GenerateHash(summary string, timestamp int64) string {
	return util.GenerateHash(summary, timestamp)
}
