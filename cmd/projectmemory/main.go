package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/localrivet/projectmemory/internal/contextstore"
	"github.com/localrivet/projectmemory/internal/logger"
	"github.com/localrivet/projectmemory/internal/server"
	"github.com/localrivet/projectmemory/internal/summarizer"
	"github.com/localrivet/projectmemory/internal/vector"
)

// Config represents the configuration structure for the ProjectMemory service.
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

func main() {
	// Initialize logging first thing
	appLogger := setupLogging()

	appLogger.Info("ProjectMemory MCP Server - Starting...")

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		logger.LogError(err)
		appLogger.Fatal("Failed to load configuration")
	}

	// Configure logging based on config
	if config.Logging.Level != "" {
		appLogger.SetLevel(logger.ParseLevel(config.Logging.Level))
		appLogger.Info("Log level set to %s", config.Logging.Level)
	}

	if config.Logging.Format == "json" {
		appLogger.SetFormat(logger.JSON)
		appLogger.Info("Log format set to JSON")
	}

	// Initialize the context store
	store := contextstore.NewSQLiteContextStore()
	storeLogger := appLogger.WithContext("store")

	err = store.Initialize(config.Database.Path)
	if err != nil {
		err = logger.DatabaseError(err, "Failed to initialize SQLite context store")
		logger.LogError(err)
		appLogger.Fatal("Failed to initialize SQLite context store")
	}
	defer store.Close()
	storeLogger.Info("SQLite context store initialized")

	// Initialize the summarizer
	summ := summarizer.NewBasicSummarizer(summarizer.DefaultMaxSummaryLength)
	summLogger := appLogger.WithContext("summarizer")

	err = summ.Initialize()
	if err != nil {
		err = logger.ConfigError(err, "Failed to initialize summarizer")
		logger.LogError(err)
		appLogger.Fatal("Failed to initialize summarizer")
	}
	summLogger.Info("Summarizer initialized")

	// Initialize the embedder
	emb := vector.NewMockEmbedder(vector.DefaultEmbeddingDimensions)
	embLogger := appLogger.WithContext("embedder")

	err = emb.Initialize()
	if err != nil {
		err = logger.ConfigError(err, "Failed to initialize embedder")
		logger.LogError(err)
		appLogger.Fatal("Failed to initialize embedder")
	}
	embLogger.Info("Embedder initialized")

	// Initialize the MCP server
	srv := server.NewContextToolServer(store, summ, emb)
	srvLogger := appLogger.WithContext("server")

	err = srv.Initialize()
	if err != nil {
		err = logger.ConfigError(err, "Failed to initialize MCP server")
		logger.LogError(err)
		appLogger.Fatal("Failed to initialize MCP server")
	}
	srvLogger.Info("MCP server initialized")

	// Handle graceful shutdown
	setupSignalHandler(store, appLogger)

	// Start the MCP server (this will block until server is terminated)
	srvLogger.Info("Starting MCP server...")
	if err := srv.Start(); err != nil {
		err = logger.APIError(err, "MCP server failed")
		logger.LogError(err)
		appLogger.Fatal("Failed to start MCP server")
	}
}

// setupLogging configures and returns the application logger
func setupLogging() *logger.Logger {
	// Create default configuration
	config := logger.DefaultConfig()

	// Try to get log level from environment variable
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		config.Level = logger.ParseLevel(levelStr)
	}

	// Create and return logger
	appLogger := logger.New(config)
	logger.SetDefaultLogger(appLogger)

	return appLogger
}

// loadConfig loads configuration from the .projectmemoryconfig file.
func loadConfig() (*Config, error) {
	data, err := os.ReadFile(".projectmemoryconfig")
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

// setupSignalHandler sets up a signal handler for graceful shutdown.
func setupSignalHandler(store contextstore.ContextStore, log *logger.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("Received shutdown signal, terminating gracefully...")

		// Close the store to ensure all data is saved
		if err := store.Close(); err != nil {
			err = logger.DatabaseError(err, "Error closing store during shutdown")
			logger.LogError(err)
		} else {
			log.Info("Database closed successfully")
		}

		log.Info("Shutdown complete")
		os.Exit(0)
	}()
}
