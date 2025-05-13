package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/localrivet/configurator"
	"github.com/localrivet/gomcp/logx"
)

// Global configuration instance
var (
	// Global is the global configuration instance
	Global *Config
	// initOnce ensures initialization happens only once
	initOnce sync.Once
)

// InitGlobal initializes the global configuration
func InitGlobal(configPath string) (*Config, error) {
	var err error
	initOnce.Do(func() {
		Global, err = LoadConfigWithPath(configPath)
	})
	return Global, err
}

// Config represents the ProjectMemory configuration
type Config struct {
	// Store contains storage-related configuration.
	Store struct {
		// SQLitePath is the path to the SQLite database file.
		SQLitePath string `json:"sqlite_path" env:"SQLITE_PATH" validate:"required"`
	} `json:"store"`

	// Summarizer contains summarization-related configuration.
	Summarizer struct {
		// Provider is the name of the summarization provider to use.
		Provider string `json:"provider" env:"SUMMARIZER_PROVIDER"`

		// ApiKey is the API key for the summarization provider.
		ApiKey string `json:"api_key" env:"SUMMARIZER_API_KEY"`
	} `json:"summarizer"`

	// Embedder contains embedding-related configuration.
	Embedder struct {
		// Provider is the name of the embedding provider to use.
		Provider string `json:"provider" env:"EMBEDDER_PROVIDER"`

		// Dimensions is the number of dimensions for the embeddings.
		Dimensions int `json:"dimensions" env:"EMBEDDER_DIMENSIONS" validate:"min:1"`

		// ApiKey is the API key for the embedding provider.
		ApiKey string `json:"api_key" env:"EMBEDDER_API_KEY"`
	} `json:"embedder"`

	// Logging contains logging-related configuration.
	Logging struct {
		// Level is the minimum log level to display ("debug", "info", "warn", "error").
		Level string `json:"level" env:"LOG_LEVEL" validate:"required"`

		// Format is the log format to use ("text", "json").
		Format string `json:"format" env:"LOG_FORMAT"`
	} `json:"logging"`

	// Internal state (not saved to config file)
	configPath     string       `json:"-"`
	mutex          sync.RWMutex `json:"-"`
	lastModifiedAt time.Time    `json:"-"`
}

// Default configuration values
const (
	DefaultConfigFilename = ".projectmemoryconfig"
	DefaultSQLitePath     = ".projectmemory.db"
	DefaultLogLevel       = "info"
	DefaultLogFormat      = "text"
)

// NewConfig creates a new Config instance with default values
func NewConfig() *Config {
	config := &Config{}
	config.Store.SQLitePath = DefaultSQLitePath
	config.Summarizer.Provider = "basic"
	config.Embedder.Provider = "mock"
	config.Embedder.Dimensions = 768 // Using a common embedding dimension
	config.Logging.Level = DefaultLogLevel
	config.Logging.Format = DefaultLogFormat
	return config
}

// LoadConfig loads the configuration from the default path
func LoadConfig() (*Config, error) {
	return LoadConfigWithPath(DefaultConfigFilename)
}

// LoadConfigWithPath loads the configuration from a specific path
func LoadConfigWithPath(configPath string) (*Config, error) {
	// Create a default logger for configuration loading
	stdLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create default configuration
	cfg := NewConfig()

	// Try to find config file if path is default
	if configPath == DefaultConfigFilename {
		foundPath, err := configurator.FindConfigFile(configPath)
		if err == nil {
			configPath = foundPath
			stdLogger.Debug("Found config file at " + foundPath)
		}
	}

	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// File doesn't exist, return default config
		stdLogger.Info("Config file not found, using default configuration", "path", configPath)
		cfg.configPath = configPath
		cfg.lastModifiedAt = time.Now()
		return cfg, nil
	}

	stdLogger.Info("Loading configuration", "path", configPath)

	// Create configurator instance
	config := configurator.New(stdLogger).
		WithProvider(configurator.NewDefaultProvider()).
		WithProvider(configurator.NewFileProvider(configPath)).
		WithProvider(configurator.NewEnvProvider("PROJECTMEMORY")).
		WithValidator(configurator.NewDefaultValidator())

	// Load configuration
	ctx := context.Background()
	if err := config.Load(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Store the config path for future operations
	cfg.configPath = configPath
	cfg.lastModifiedAt = time.Now()

	return cfg, nil
}

// SaveToFile saves the configuration to the specified file
func (c *Config) SaveToFile(path string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save using configurator's SaveToFile function
	if err := configurator.SaveToFile(c, path, configurator.FormatJSON); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Update internal state
	c.configPath = path
	c.lastModifiedAt = time.Now()

	return nil
}

// Save saves the configuration to the last used file path
func (c *Config) Save() error {
	if c.configPath == "" {
		c.configPath = DefaultConfigFilename
	}
	return c.SaveToFile(c.configPath)
}

// GetConfigPath returns the path of the currently loaded configuration file
func (c *Config) GetConfigPath() string {
	return c.configPath
}

// ToLegacyConfig converts the internal config to the format used in projectmemory.go
// This is for backward compatibility
func (c *Config) ToLegacyConfig() map[string]interface{} {
	return map[string]interface{}{
		"store": map[string]interface{}{
			"sqlite_path": c.Store.SQLitePath,
		},
		"summarizer": map[string]interface{}{
			"provider": c.Summarizer.Provider,
			"api_key":  c.Summarizer.ApiKey,
		},
		"embedder": map[string]interface{}{
			"provider":   c.Embedder.Provider,
			"dimensions": c.Embedder.Dimensions,
			"api_key":    c.Embedder.ApiKey,
		},
		"logging": map[string]interface{}{
			"level":  c.Logging.Level,
			"format": c.Logging.Format,
		},
	}
}

// GetLoggerFromConfig creates a gomcp logx.Logger based on the configuration
func GetLoggerFromConfig(cfg *Config) logx.Logger {
	return logx.NewLogger(cfg.Logging.Level)
}
