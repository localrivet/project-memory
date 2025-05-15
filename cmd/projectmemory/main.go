package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/localrivet/projectmemory"
	"github.com/localrivet/projectmemory/internal/config"
	"github.com/localrivet/projectmemory/internal/contextstore"
)

const (
	defaultConfigPath = ".projectmemoryconfig"
)

var programLevel = new(slog.LevelVar)

func main() {
	// Set up logging with slog
	setupSlog()

	// Get configuration path from arguments or use default
	configPath := defaultConfigPath
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	slog.Info("ProjectMemory MCP Server - Starting...")

	// Check if config file exists before trying to create server
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		slog.Info("Configuration file not found", "path", configPath)

		// Ask if user wants to create default config
		if promptCreateConfig(configPath) {
			slog.Info("Creating default configuration file", "path", configPath)

			// Create default config
			cfg := config.NewConfig()
			if err := cfg.SaveToFile(configPath); err != nil {
				slog.Error("Failed to create default configuration", "error", err)
				os.Exit(1)
			}

			slog.Info("Default configuration file created successfully")
		} else {
			slog.Info("Configuration file creation skipped. Exiting.")
			os.Exit(0)
		}
	}

	// Create the server
	server, err := projectmemory.NewServer(projectmemory.ServerOptions{
		ConfigPath: configPath,
		// Let it use slog.Default() for logging (set up in setupSlog)
	})
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Initialize components
	store, err := initStore()
	if err != nil {
		slog.Error("Failed to initialize SQLite context store", "error", err)
		os.Exit(1)
	}
	slog.Info("SQLite context store initialized")

	// Set up signal handler for graceful shutdown
	setupSignalHandler(store)

	// Start the server
	slog.Info("Starting MCP server...")
	err = server.Start()
	if err != nil {
		slog.Error("Failed to start MCP server", "error", err)
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
	fmt.Fprint(os.Stdout, "Configuration file not found. Create default configuration? [Y/n]: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		// Error reading input, assume yes
		return true
	}

	response = strings.TrimSpace(strings.ToLower(response))
	// If response is empty or starts with 'y', return true
	return response == "" || strings.HasPrefix(response, "y")
}

func setupSlog() {
	logLevelStr := os.Getenv("LOG_LEVEL")
	var level slog.Level
	switch strings.ToLower(logLevelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	programLevel.Set(level)

	var handler slog.Handler
	logFormat := os.Getenv("LOG_FORMAT")
	logOutput := os.Getenv("PROJECTMEMORY_LOG_OUTPUT")

	var outputWriter io.Writer
	if strings.ToLower(logOutput) == "discard" {
		outputWriter = io.Discard
		slog.Info("Logging disabled for MCP stdio mode. All log output will be discarded.")
	} else {
		outputWriter = os.Stderr
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     programLevel,
	}

	// Temporary logger to report initial settings if not discarding
	var initialLogger *slog.Logger
	if outputWriter != io.Discard {
		if strings.ToLower(logFormat) == "json" {
			initialLogger = slog.New(slog.NewJSONHandler(outputWriter, opts))
		} else {
			initialLogger = slog.New(slog.NewTextHandler(outputWriter, opts))
		}
		initialLogger.Info("Logging output to stderr")
	}

	if strings.ToLower(logFormat) == "json" {
		handler = slog.NewJSONHandler(outputWriter, opts)
	} else {
		handler = slog.NewTextHandler(outputWriter, opts)
	}
	slog.SetDefault(slog.New(handler))

	// Log final settings using the now default logger
	slog.Info("Logging initialized", "level", programLevel.Level().String(), "format", logFormat, "output_target", logOutput)

	// If discarding, explicitly state it again now that default logger is set to discard
	if outputWriter == io.Discard {
		// This message will actually be discarded. We rely on the initial pre-SetDefault message.
		// Consider logging important startup messages before setting handler to io.Discard if they must be seen.
	}
}

func initStore() (contextstore.ContextStore, error) {
	// Get database path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = config.DefaultSQLitePath // Use the default from config package
	}

	// Initialize SQLite store
	slog.Info("Initializing SQLite store", "path", dbPath)
	store := contextstore.NewSQLiteContextStore()
	err := store.Initialize(dbPath)
	if err != nil {
		slog.Error("Failed to initialize SQLite context store", "error", err, "path", dbPath)
		return nil, err
	}

	return store, nil
}

func setupSignalHandler(store contextstore.ContextStore) {
	// Create channel to receive signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Handle signals in a goroutine
	go func() {
		<-c
		slog.Info("Shutting down gracefully...")

		// Close the store
		err := store.Close()
		if err != nil {
			slog.Error("Error closing store during shutdown", "error", err)
		}

		os.Exit(0)
	}()
}
