package projectmemory

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

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
	logger     *slog.Logger // Logger for this Server instance
}

// ServerOptions defines the options for creating a new Server.
type ServerOptions struct {
	Config     *Config      // Pre-filled config. If nil, ConfigPath is used.
	ConfigPath string       // Path to config file. Used if Config is nil. If both are empty, DefaultConfig() is used.
	Logger     *slog.Logger // External logger. If nil, slog.Default() is used.
}

// NewServer creates a new ProjectMemory Server with the given options.
// If opts.Config is provided, it will be used directly.
// Otherwise, if opts.ConfigPath is provided, configuration will be loaded from that path.
// If neither is provided, DefaultConfig() will be used.
// If opts.Logger is nil, slog.Default() will be used.
func NewServer(opts ServerOptions) (*Server, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	var cfg *Config
	var err error

	if opts.Config != nil {
		cfg = opts.Config
		logger.Info("Using provided Config object for server initialization")
	} else if opts.ConfigPath != "" {
		logger.Info("Loading configuration for server initialization", "path", opts.ConfigPath)
		cfg, err = config.LoadConfigWithPath(opts.ConfigPath)
		if err != nil {
			logger.Error("Failed to load configuration from path", "path", opts.ConfigPath, "error", err)
			return nil, errortypes.ConfigError(err, "Failed to load configuration from path: "+opts.ConfigPath)
		}
	} else {
		logger.Warn("No Config object or ConfigPath provided, using default configuration for server initialization")
		cfg = DefaultConfig()
	}

	store, sum, emb, err := CreateComponents(cfg, logger) // Pass logger to CreateComponents
	if err != nil {
		// CreateComponents already logs the specific error
		logger.Error("Failed to create components during server initialization", "error", err)
		return nil, err // Return the original error which should be specific enough
	}

	logger.Info("Initializing context tool server component")
	mcpServer := server.NewContextToolServer(store, sum, emb)
	err = mcpServer.Initialize() // Note: mcpServer.Initialize still uses global slog internally
	if err != nil {
		logger.Error("Failed to initialize MCP context tool server component", "error", err)
		return nil, errortypes.ConfigError(err, "Failed to initialize MCP context tool server component")
	}

	logger.Info("ProjectMemory server successfully initialized")
	return &Server{
		config:     cfg,
		store:      store,
		summarizer: sum,
		embedder:   emb,
		toolServer: mcpServer,
		logger:     logger, // Store the resolved logger
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
		// The Stop method of toolServer might return an error that should be logged.
		s.logger.Error("Error stopping tool server", "error", err)
		return err
	}

	// Close the store
	s.logger.Info("Closing store")
	err = s.store.Close()
	if err != nil {
		s.logger.Error("Failed to close store", "error", err)
		return err
	}

	s.logger.Info("ProjectMemory service stopped")
	return nil
}

// SaveContext saves the given text to the context store.
func (s *Server) SaveContext(text string) (string, error) {
	// Generate summary
	s.logger.Debug("Generating summary of text", "length", len(text))
	summary, err := s.summarizer.Summarize(text)
	if err != nil {
		s.logger.Error("Failed to summarize text", "error", err)
		return "", err
	}

	// Create embedding
	s.logger.Debug("Creating embedding for summary")
	embedding, err := s.embedder.CreateEmbedding(summary)
	if err != nil {
		s.logger.Error("Failed to create embedding", "error", err)
		return "", err
	}

	// Convert embedding to bytes
	embeddingBytes, err := vector.Float32SliceToBytes(embedding)
	if err != nil {
		s.logger.Error("Failed to convert embedding to bytes", "error", err)
		return "", err
	}

	// Generate ID (simple hash of content + timestamp)
	timestamp := time.Now()
	id := GenerateHash(summary, timestamp.UnixNano())

	// Store in context store
	s.logger.Debug("Storing context", "id", id)
	err = s.store.Store(id, summary, embeddingBytes, timestamp)
	if err != nil {
		s.logger.Error("Failed to store context", "id", id, "error", err)
		return "", err
	}

	s.logger.Info("Successfully saved context", "id", id)
	return id, nil
}

// RetrieveContext retrieves context entries similar to the given query.
func (s *Server) RetrieveContext(query string, limit int) ([]string, error) {
	// Create embedding for query
	s.logger.Debug("Creating embedding for query", "query", query)
	queryEmbedding, err := s.embedder.CreateEmbedding(query)
	if err != nil {
		s.logger.Error("Failed to create embedding for query", "query", query, "error", err)
		return nil, err
	}

	// Search context store
	s.logger.Debug("Searching for similar context entries", "limit", limit)
	results, err := s.store.Search(queryEmbedding, limit)
	if err != nil {
		s.logger.Error("Failed to search context store", "limit", limit, "error", err)
		return nil, err
	}

	s.logger.Info("Retrieved context entries", "count", len(results))
	return results, nil
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
func CreateComponents(cfg *Config, logger *slog.Logger) (contextstore.ContextStore, summarizer.Summarizer, vector.Embedder, error) {
	if logger == nil {
		// This case should ideally not be hit if NewServerWithOptions always provides one,
		// but as a public function, it's safer to have a fallback.
		logger = slog.Default()
		logger.Debug("CreateComponents called with nil logger, defaulting to slog.Default()")
	}

	// Initialize SQLite context store
	logger.Info("Initializing SQLite context store for CreateComponents", "path", cfg.Store.SQLitePath)
	store := contextstore.NewSQLiteContextStore()
	err := store.Initialize(cfg.Store.SQLitePath)
	if err != nil {
		logger.Error("Failed to initialize SQLite context store in CreateComponents", "path", cfg.Store.SQLitePath, "error", err)
		return nil, nil, nil, errortypes.DatabaseError(err, "Failed to initialize SQLite context store")
	}

	// Initialize summarizer
	logger.Info("Initializing summarizer for CreateComponents", "provider", cfg.Summarizer.Provider)
	var sum summarizer.Summarizer
	switch cfg.Summarizer.Provider {
	case "basic", "":
		sum = summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	default:
		logger.Warn("Unknown summarizer provider in CreateComponents, using basic summarizer", "provider", cfg.Summarizer.Provider)
		sum = summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	}

	if err := sum.Initialize(); err != nil {
		logger.Error("Failed to initialize summarizer in CreateComponents", "error", err)
		return nil, nil, nil, errortypes.ConfigError(err, "Failed to initialize summarizer")
	}

	// Initialize embedder
	logger.Info("Initializing embedder for CreateComponents", "provider", cfg.Embedder.Provider, "dimensions", cfg.Embedder.Dimensions)
	var emb vector.Embedder
	dimensions := cfg.Embedder.Dimensions
	if dimensions <= 0 {
		dimensions = vector.DefaultEmbeddingDimensions
	}

	switch cfg.Embedder.Provider {
	case "mock", "":
		emb = vector.NewMockEmbedder(dimensions)
	default:
		logger.Warn("Unknown embedder provider in CreateComponents, using mock embedder", "provider", cfg.Embedder.Provider)
		emb = vector.NewMockEmbedder(dimensions)
	}

	if err := emb.Initialize(); err != nil {
		logger.Error("Failed to initialize embedder in CreateComponents", "error", err)
		return nil, nil, nil, errortypes.ConfigError(err, "Failed to initialize embedder")
	}

	logger.Info("Components successfully initialized via CreateComponents")
	return store, sum, emb, nil
}

// GenerateHash creates a hash from the summary and a timestamp
// This is a convenience wrapper around the internal util.GenerateHash function
func GenerateHash(summary string, timestamp int64) string {
	return util.GenerateHash(summary, timestamp)
}
