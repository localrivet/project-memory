package summarizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/localrivet/project-memory/internal/summarizer/providers"
	"github.com/localrivet/project-memory/internal/telemetry"
)

const (
	// Default settings
	DefaultTimeout       = 30 * time.Second
	DefaultMaxRetries    = 3
	DefaultRetryDelay    = 2 * time.Second
	DefaultCacheCapacity = 1000
	DefaultCacheTTL      = 24 * time.Hour
)

// Errors
var (
	ErrProviderNotSupported = errors.New("provider not supported")
	ErrSummarizationFailed  = errors.New("summarization failed")
	ErrConfigError          = errors.New("configuration error")
	ErrContextCanceled      = errors.New("context canceled")
)

// Using providers.LLMProvider instead of a local definition

// AISummarizer is an implementation of the Summarizer interface
// that uses LLMs to create high-quality summaries
type AISummarizer struct {
	provider            providers.LLMProvider
	fallbackProviders   []providers.LLMProvider
	maxSummaryLength    int
	timeout             time.Duration
	maxRetries          int
	retryDelay          time.Duration
	cache               *summaryCache
	httpClient          *http.Client
	providerInitialized bool
	providerFactory     *providers.ProviderFactory
	metrics             *telemetry.MetricsCollector
	mu                  sync.RWMutex
}

// summaryCache provides thread-safe caching for summaries
type summaryCache struct {
	items    map[string]cachedSummary
	capacity int
	ttl      time.Duration
	mu       sync.RWMutex
}

// cachedSummary represents a cached summary with expiration
type cachedSummary struct {
	summary  string
	expireAt time.Time
}

// NewAISummarizer creates a new AISummarizer with the specified provider and settings
func NewAISummarizer(config *AISummarizerConfig) *AISummarizer {
	if config == nil {
		config = &AISummarizerConfig{}
	}

	// Set defaults if not specified
	if config.MaxSummaryLength <= 0 {
		config.MaxSummaryLength = DefaultMaxSummaryLength
	}
	if config.Timeout <= 0 {
		config.Timeout = DefaultTimeout
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = DefaultMaxRetries
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = DefaultRetryDelay
	}
	if config.CacheCapacity <= 0 {
		config.CacheCapacity = DefaultCacheCapacity
	}
	if config.CacheTTL <= 0 {
		config.CacheTTL = DefaultCacheTTL
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Create cache
	cache := &summaryCache{
		items:    make(map[string]cachedSummary),
		capacity: config.CacheCapacity,
		ttl:      config.CacheTTL,
	}

	// Create metrics collector
	metrics := telemetry.NewMetricsCollector()

	return &AISummarizer{
		maxSummaryLength: config.MaxSummaryLength,
		timeout:          config.Timeout,
		maxRetries:       config.MaxRetries,
		retryDelay:       config.RetryDelay,
		cache:            cache,
		httpClient:       httpClient,
		metrics:          metrics,
	}
}

// AISummarizerConfig holds configuration for the AISummarizer
type AISummarizerConfig struct {
	ProviderName      string
	ModelID           string
	APIKey            string
	MaxSummaryLength  int
	Timeout           time.Duration
	MaxRetries        int
	RetryDelay        time.Duration
	CacheCapacity     int
	CacheTTL          time.Duration
	FallbackProviders []struct {
		Name    string
		ModelID string
		APIKey  string
	}
}

// Initialize sets up the summarizer with required configuration
func (s *AISummarizer) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If already initialized, do nothing
	if s.providerInitialized {
		return nil
	}

	// Create provider based on config
	if s.provider == nil {
		// Load configuration from config file and environment variables
		config, err := loadConfigFromEnvironment()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Create provider configs
		providerConfigs := make(map[string]providers.Config)

		// Add main provider
		providerConfigs[config.ProviderName] = providers.Config{
			ModelID: config.ModelID,
			APIKey:  config.APIKey,
		}

		// Add fallback providers
		for _, fallbackConfig := range config.FallbackProviders {
			providerConfigs[fallbackConfig.Name] = providers.Config{
				ModelID: fallbackConfig.ModelID,
				APIKey:  fallbackConfig.APIKey,
			}
		}

		// Create provider factory
		s.providerFactory = providers.NewProviderFactory(providerConfigs)

		// Create primary provider
		primaryProvider, err := s.providerFactory.GetProvider(config.ProviderName)
		if err != nil {
			return fmt.Errorf("failed to create primary provider: %w", err)
		}
		s.provider = primaryProvider

		// Create fallback provider chain
		// First try with explicit fallback order
		var preferenceOrder []string
		for _, fb := range config.FallbackProviders {
			preferenceOrder = append(preferenceOrder, fb.Name)
		}

		s.fallbackProviders = s.providerFactory.GetProviderChain(preferenceOrder)
	}

	s.providerInitialized = true
	return nil
}

// loadConfigFromEnvironment loads configuration from environment variables
func loadConfigFromEnvironment() (*AISummarizerConfig, error) {
	// Get the primary provider configuration
	primaryProvider := getEnvWithDefault("AI_SUMMARIZER_PROVIDER", providers.ProviderAnthropic)
	primaryModelID := getEnvWithDefault("AI_SUMMARIZER_MODEL_ID", "")
	primaryAPIKey := getProviderAPIKey(primaryProvider)

	if primaryAPIKey == "" {
		return nil, fmt.Errorf("%w: missing API key for primary provider %s", ErrConfigError, primaryProvider)
	}

	// Parse numeric settings with defaults
	maxSummaryLen := getEnvIntWithDefault("AI_SUMMARIZER_MAX_LENGTH", DefaultMaxSummaryLength)
	maxRetries := getEnvIntWithDefault("AI_SUMMARIZER_MAX_RETRIES", DefaultMaxRetries)
	cacheCapacity := getEnvIntWithDefault("AI_SUMMARIZER_CACHE_CAPACITY", DefaultCacheCapacity)

	// Parse duration settings with defaults
	timeout := getEnvDurationWithDefault("AI_SUMMARIZER_TIMEOUT", DefaultTimeout)
	retryDelay := getEnvDurationWithDefault("AI_SUMMARIZER_RETRY_DELAY", DefaultRetryDelay)
	cacheTTL := getEnvDurationWithDefault("AI_SUMMARIZER_CACHE_TTL", DefaultCacheTTL)

	// Build the configuration
	config := &AISummarizerConfig{
		ProviderName:     primaryProvider,
		ModelID:          primaryModelID,
		APIKey:           primaryAPIKey,
		MaxSummaryLength: maxSummaryLen,
		Timeout:          timeout,
		MaxRetries:       maxRetries,
		RetryDelay:       retryDelay,
		CacheCapacity:    cacheCapacity,
		CacheTTL:         cacheTTL,
	}

	// Get fallback provider order
	fallbackOrder := getEnvWithDefault("AI_SUMMARIZER_FALLBACK_ORDER", "openai,google,xai")
	fallbackProviders := strings.Split(fallbackOrder, ",")

	// Configure each fallback provider
	for _, providerName := range fallbackProviders {
		providerName = strings.TrimSpace(providerName)
		if providerName == primaryProvider || providerName == "" {
			continue
		}

		apiKey := getProviderAPIKey(providerName)
		if apiKey == "" {
			// Skip providers with no API key
			continue
		}

		// Get model ID for this provider, if specified
		modelIDEnvVar := fmt.Sprintf("AI_SUMMARIZER_%s_MODEL_ID", strings.ToUpper(providerName))
		modelID := getEnvWithDefault(modelIDEnvVar, "")

		config.FallbackProviders = append(config.FallbackProviders, struct {
			Name    string
			ModelID string
			APIKey  string
		}{
			Name:    providerName,
			ModelID: modelID,
			APIKey:  apiKey,
		})
	}

	return config, nil
}

// getProviderAPIKey retrieves the API key for the specified provider
func getProviderAPIKey(providerName string) string {
	switch providerName {
	case providers.ProviderAnthropic:
		return os.Getenv("ANTHROPIC_API_KEY")
	case providers.ProviderOpenAI:
		return os.Getenv("OPENAI_API_KEY")
	case providers.ProviderGoogle:
		return os.Getenv("GOOGLE_API_KEY")
	case providers.ProviderXAI:
		return os.Getenv("XAI_API_KEY")
	default:
		return ""
	}
}

// getEnvWithDefault retrieves an environment variable or returns the default value
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvIntWithDefault retrieves an environment variable as int or returns the default value
func getEnvIntWithDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvDurationWithDefault retrieves an environment variable as duration or returns the default value
func getEnvDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// Summarize takes a text input and returns a condensed summary using LLMs
func (s *AISummarizer) Summarize(text string) (string, error) {
	startTime := time.Now()
	defer func() {
		s.metrics.RecordTimer("summarizer.total_time", time.Since(startTime))
	}()

	s.mu.RLock()
	if !s.providerInitialized {
		s.mu.RUnlock()
		if err := s.Initialize(); err != nil {
			return "", fmt.Errorf("failed to initialize summarizer: %w", err)
		}
	} else {
		s.mu.RUnlock()
	}

	// Check cache first
	if summary, found := s.checkCache(text); found {
		s.metrics.IncrementCounter(telemetry.MetricCacheHits, 1)
		return summary, nil
	}
	s.metrics.IncrementCounter(telemetry.MetricCacheMisses, 1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// Track current provider for metrics
	var currentProviderMetric string
	if s.provider != nil {
		switch s.provider.Name() {
		case providers.ProviderAnthropic:
			currentProviderMetric = telemetry.MetricAPICallsAnthropic
		case providers.ProviderOpenAI:
			currentProviderMetric = telemetry.MetricAPICallsOpenAI
		case providers.ProviderGoogle:
			currentProviderMetric = telemetry.MetricAPICallsGoogle
		case providers.ProviderXAI:
			currentProviderMetric = telemetry.MetricAPICallsXAI
		}
		s.metrics.IncrementCounter(currentProviderMetric, 1)
	}

	// Try with primary provider with retries
	primaryStart := time.Now()
	summary, err := s.summarizeWithRetries(ctx, text)
	if err == nil {
		// Cache the successful result
		s.cacheResult(text, summary)
		s.metrics.IncrementCounter(telemetry.MetricAPICallsSuccess, 1)

		// Record response time for the provider
		switch s.provider.Name() {
		case providers.ProviderAnthropic:
			s.metrics.RecordTimer(telemetry.MetricResponseTimeAnthropic, time.Since(primaryStart))
		case providers.ProviderOpenAI:
			s.metrics.RecordTimer(telemetry.MetricResponseTimeOpenAI, time.Since(primaryStart))
		case providers.ProviderGoogle:
			s.metrics.RecordTimer(telemetry.MetricResponseTimeGoogle, time.Since(primaryStart))
		case providers.ProviderXAI:
			s.metrics.RecordTimer(telemetry.MetricResponseTimeXAI, time.Since(primaryStart))
		}

		return summary, nil
	}

	// Record primary provider failure
	s.metrics.IncrementCounter(telemetry.MetricAPICallsFailure, 1)
	s.metrics.IncrementCounter(telemetry.MetricFallbackAttempts, 1)

	// If primary provider fails, try fallbacks
	for _, fallbackProvider := range s.fallbackProviders {
		ctx, cancel = context.WithTimeout(context.Background(), s.timeout)
		tempProvider := s.provider    // Save current provider
		s.provider = fallbackProvider // Temporarily switch provider

		// Track current fallback provider for metrics
		switch fallbackProvider.Name() {
		case providers.ProviderAnthropic:
			s.metrics.IncrementCounter(telemetry.MetricAPICallsAnthropic, 1)
		case providers.ProviderOpenAI:
			s.metrics.IncrementCounter(telemetry.MetricAPICallsOpenAI, 1)
		case providers.ProviderGoogle:
			s.metrics.IncrementCounter(telemetry.MetricAPICallsGoogle, 1)
		case providers.ProviderXAI:
			s.metrics.IncrementCounter(telemetry.MetricAPICallsXAI, 1)
		}

		fallbackStart := time.Now()
		summary, err = s.summarizeWithRetries(ctx, text)
		s.provider = tempProvider // Restore original provider
		cancel()

		if err == nil {
			// Cache the successful result
			s.cacheResult(text, summary)
			s.metrics.IncrementCounter(telemetry.MetricAPICallsSuccess, 1)
			s.metrics.IncrementCounter(telemetry.MetricFallbackSuccess, 1)

			// Record response time for the fallback provider
			switch fallbackProvider.Name() {
			case providers.ProviderAnthropic:
				s.metrics.RecordTimer(telemetry.MetricResponseTimeAnthropic, time.Since(fallbackStart))
			case providers.ProviderOpenAI:
				s.metrics.RecordTimer(telemetry.MetricResponseTimeOpenAI, time.Since(fallbackStart))
			case providers.ProviderGoogle:
				s.metrics.RecordTimer(telemetry.MetricResponseTimeGoogle, time.Since(fallbackStart))
			case providers.ProviderXAI:
				s.metrics.RecordTimer(telemetry.MetricResponseTimeXAI, time.Since(fallbackStart))
			}

			return summary, nil
		}

		// Record fallback provider failure
		s.metrics.IncrementCounter(telemetry.MetricAPICallsFailure, 1)
	}

	// If all providers fail, use BasicSummarizer as final fallback
	basicSummarizer := NewBasicSummarizer(s.maxSummaryLength)
	summary, err = basicSummarizer.Summarize(text)
	if err != nil {
		return "", ErrSummarizationFailed
	}

	// Cache the fallback result
	s.cacheResult(text, summary)
	return summary, nil
}

// summarizeWithRetries attempts to summarize text with the current provider, with retries
func (s *AISummarizer) summarizeWithRetries(ctx context.Context, text string) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		// Check if context is canceled before making the attempt
		select {
		case <-ctx.Done():
			return "", ErrContextCanceled
		default:
		}

		if attempt > 0 {
			// Track retry attempts
			s.metrics.IncrementCounter(telemetry.MetricRetryAttempts, 1)

			// Wait before retry with exponential backoff
			retryDelay := s.retryDelay * time.Duration(attempt)
			time.Sleep(retryDelay)
		}

		summary, err := s.provider.Summarize(ctx, text, s.maxSummaryLength)
		if err == nil {
			if attempt > 0 {
				// Track successful retry
				s.metrics.IncrementCounter(telemetry.MetricRetrySuccess, 1)
			}
			return summary, nil
		}

		lastErr = err
	}

	return "", lastErr
}

// checkCache looks for a cached summary
func (s *AISummarizer) checkCache(text string) (string, bool) {
	// Create a proper hash of the text as the key
	hash := sha256.Sum256([]byte(text))
	key := hex.EncodeToString(hash[:])

	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	if item, exists := s.cache.items[key]; exists {
		// Check if the cached item is still valid
		if time.Now().Before(item.expireAt) {
			return item.summary, true
		}
	}

	return "", false
}

// cacheResult stores a summary in the cache
func (s *AISummarizer) cacheResult(text, summary string) {
	// Create a proper hash of the text as the key
	hash := sha256.Sum256([]byte(text))
	key := hex.EncodeToString(hash[:])

	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Enforce cache capacity by evicting oldest items if needed
	if len(s.cache.items) >= s.cache.capacity {
		// Simple eviction strategy - delete a random item
		// In a real implementation, use LRU or similar policy
		for k := range s.cache.items {
			delete(s.cache.items, k)
			break
		}
	}

	// Store the new item
	s.cache.items[key] = cachedSummary{
		summary:  summary,
		expireAt: time.Now().Add(s.cache.ttl),
	}

	// Update cache size metric
	s.metrics.SetGauge(telemetry.MetricCacheSize, float64(len(s.cache.items)))
}

// GetMetrics returns the metrics collector for this summarizer
func (s *AISummarizer) GetMetrics() *telemetry.MetricsCollector {
	return s.metrics
}

// CheckProviderHealth tests if all providers are operational
func (s *AISummarizer) CheckProviderHealth() map[string]bool {
	results := make(map[string]bool)
	testText := "This is a brief health check for the LLM provider."

	// First, ensure the AISummarizer is initialized
	if err := s.Initialize(); err != nil {
		return results
	}

	// Check primary provider
	if s.provider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := s.provider.Summarize(ctx, testText, 50)
		cancel()

		providerName := s.provider.Name()
		results[providerName] = (err == nil)

		// Record health status
		switch providerName {
		case providers.ProviderAnthropic:
			s.metrics.SetGauge(telemetry.MetricProviderHealthAnthropic, boolToFloat64(results[providerName]))
		case providers.ProviderOpenAI:
			s.metrics.SetGauge(telemetry.MetricProviderHealthOpenAI, boolToFloat64(results[providerName]))
		case providers.ProviderGoogle:
			s.metrics.SetGauge(telemetry.MetricProviderHealthGoogle, boolToFloat64(results[providerName]))
		case providers.ProviderXAI:
			s.metrics.SetGauge(telemetry.MetricProviderHealthXAI, boolToFloat64(results[providerName]))
		}
	}

	// Check fallback providers
	for _, provider := range s.fallbackProviders {
		providerName := provider.Name()
		if _, alreadyChecked := results[providerName]; alreadyChecked {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := provider.Summarize(ctx, testText, 50)
		cancel()

		results[providerName] = (err == nil)

		// Record health status
		switch providerName {
		case providers.ProviderAnthropic:
			s.metrics.SetGauge(telemetry.MetricProviderHealthAnthropic, boolToFloat64(results[providerName]))
		case providers.ProviderOpenAI:
			s.metrics.SetGauge(telemetry.MetricProviderHealthOpenAI, boolToFloat64(results[providerName]))
		case providers.ProviderGoogle:
			s.metrics.SetGauge(telemetry.MetricProviderHealthGoogle, boolToFloat64(results[providerName]))
		case providers.ProviderXAI:
			s.metrics.SetGauge(telemetry.MetricProviderHealthXAI, boolToFloat64(results[providerName]))
		}
	}

	return results
}

// boolToFloat64 converts a boolean to a float64 (1.0 for true, 0.0 for false)
func boolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
