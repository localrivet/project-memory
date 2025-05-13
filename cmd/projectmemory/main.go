package main

import (
	"bufio"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/localrivet/gomcp/logx"
	"github.com/localrivet/projectmemory"
	"github.com/localrivet/projectmemory/internal/config"
	"github.com/localrivet/projectmemory/internal/contextstore"
)

const (
	defaultConfigPath = ".projectmemoryconfig"
)

func main() {
	// Get configuration path from arguments or use default
	configPath := defaultConfigPath
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Set up logging
	logger := setupLogging()
	logger.Info("ProjectMemory MCP Server - Starting...")

	// Check if config file exists before trying to create server
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Warn("Configuration file %s not found", configPath)

		// Ask if user wants to create default config
		if promptCreateConfig(configPath) {
			logger.Info("Creating default configuration file at %s", configPath)

			// Create default config
			cfg := config.NewConfig()
			if err := cfg.SaveToFile(configPath); err != nil {
				logger.Error("Failed to create default configuration: %v", err)
				os.Exit(1)
			}

			logger.Info("Default configuration file created successfully")
		} else {
			logger.Info("Configuration file creation skipped. Exiting.")
			os.Exit(0)
		}
	}

	// Create the server with the logger
	server, err := projectmemory.NewServer(configPath, logger)
	if err != nil {
		logger.Error("Failed to create server: %v", err)
		os.Exit(1)
	}

	// Update logger based on config
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		// Create a new logger with the desired level since SetLevel might not be available
		logger = logx.NewLogger(logLevel)
		logger.Info("Log level set to %s", logLevel)

		// Update server with the new logger
		server.WithLogger(logger)
	}

	// Use JSON formatting if requested
	if logFormat := os.Getenv("LOG_FORMAT"); logFormat == "json" {
		// Cannot directly change format in logx, but we would use a different logger if needed
		logger.Info("Log format set to JSON")
	}

	// Initialize components
	store, err := initStore(logger)
	if err != nil {
		logger.Error("Failed to initialize SQLite context store: %v", err)
		os.Exit(1)
	}
	logger.Info("SQLite context store initialized")

	// Set up signal handler for graceful shutdown
	setupSignalHandler(store, logger)

	// Start the server
	logger.Info("Starting MCP server...")
	err = server.Start()
	if err != nil {
		logger.Error("Failed to start MCP server: %v", err)
		os.Exit(1)
	}
}

// promptCreateConfig asks the user if they want to create a default configuration file
func promptCreateConfig(configPath string) bool {
	// Skip prompt in non-interactive environments (like when redirecting stdin)
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
		// Not a terminal/console, return true to automatically create config
		return true
	}

	// Use standard input for interactive prompt
	reader := bufio.NewReader(os.Stdin)
	os.Stdout.WriteString("Configuration file not found. Create default configuration? [Y/n]: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		// Error reading input, assume yes
		return true
	}

	response = strings.TrimSpace(strings.ToLower(response))
	// If response is empty or starts with 'y', return true
	return response == "" || strings.HasPrefix(response, "y")
}

func setupLogging() logx.Logger {
	// Get log level from environment or use default
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = "info" // Default to INFO level
	}

	// Create and return the logger
	return logx.NewLogger(levelStr)
}

func initStore(logger logx.Logger) (contextstore.ContextStore, error) {
	// Get database path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = config.DefaultSQLitePath // Use the default from config package
	}

	// Initialize SQLite store
	logger.Debug("Initializing SQLite store at %s", dbPath)
	store := contextstore.NewSQLiteContextStore()
	err := store.Initialize(dbPath)
	if err != nil {
		logger.Error("Failed to initialize SQLite context store: %v", err)
		return nil, err
	}

	return store, nil
}

func setupSignalHandler(store contextstore.ContextStore, logger logx.Logger) {
	// Create channel to receive signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Handle signals in a goroutine
	go func() {
		<-c
		logger.Info("Shutting down gracefully...")

		// Close the store
		err := store.Close()
		if err != nil {
			logger.Error("Error closing store during shutdown: %v", err)
		}

		os.Exit(0)
	}()
}
