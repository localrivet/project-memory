package summarizer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/localrivet/projectmemory/internal/telemetry"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	// StatusHealthy indicates a component is fully operational
	StatusHealthy HealthStatus = "healthy"

	// StatusDegraded indicates a component is operational but with reduced capability
	StatusDegraded HealthStatus = "degraded"

	// StatusUnhealthy indicates a component is not operational
	StatusUnhealthy HealthStatus = "unhealthy"
)

// HealthReport contains information about the current health of the AI summarizer
type HealthReport struct {
	Status        HealthStatus       `json:"status"`
	Timestamp     time.Time          `json:"timestamp"`
	Components    map[string]string  `json:"components"`
	Providers     map[string]bool    `json:"providers"`
	ResponseTimes map[string]float64 `json:"response_times_ms"`
	CacheStats    map[string]int64   `json:"cache_stats"`
	SuccessRate   float64            `json:"success_rate"`
	TotalRequests int64              `json:"total_requests"`
	Version       string             `json:"version"`
}

// CreateHealthReport generates a health report for the AI summarizer
func CreateHealthReport(summarizer *AISummarizer) (*HealthReport, error) {
	if summarizer == nil {
		return nil, fmt.Errorf("summarizer is nil")
	}

	m := summarizer.GetMetrics()
	if m == nil {
		return nil, fmt.Errorf("metrics collector is nil")
	}

	// Check provider health
	providerHealth := summarizer.CheckProviderHealth()

	// Determine overall status
	status := StatusHealthy
	workingProviders := 0
	for _, isHealthy := range providerHealth {
		if isHealthy {
			workingProviders++
		}
	}

	if workingProviders == 0 {
		status = StatusUnhealthy
	} else if workingProviders < len(providerHealth) {
		status = StatusDegraded
	}

	// Calculate success rate
	totalSuccess := m.GetCounter(telemetry.MetricAPICallsSuccess)
	totalFailure := m.GetCounter(telemetry.MetricAPICallsFailure)
	totalRequests := totalSuccess + totalFailure

	var successRate float64
	if totalRequests > 0 {
		successRate = float64(totalSuccess) / float64(totalRequests) * 100.0
	}

	// Get response times
	responseTimes := map[string]float64{
		"anthropic": float64(m.GetTimerAverage(telemetry.MetricResponseTimeAnthropic)) / float64(time.Millisecond),
		"openai":    float64(m.GetTimerAverage(telemetry.MetricResponseTimeOpenAI)) / float64(time.Millisecond),
		"google":    float64(m.GetTimerAverage(telemetry.MetricResponseTimeGoogle)) / float64(time.Millisecond),
		"xai":       float64(m.GetTimerAverage(telemetry.MetricResponseTimeXAI)) / float64(time.Millisecond),
		"total":     float64(m.GetTimerAverage("summarizer.total_time")) / float64(time.Millisecond),
	}

	// Get cache stats
	cacheStats := map[string]int64{
		"hits":   m.GetCounter(telemetry.MetricCacheHits),
		"misses": m.GetCounter(telemetry.MetricCacheMisses),
		"size":   int64(m.GetGauge(telemetry.MetricCacheSize)),
	}

	// Create components status
	components := map[string]string{
		"cache":     string(StatusHealthy),
		"primary":   string(StatusUnhealthy),
		"fallbacks": string(StatusUnhealthy),
	}

	// Update component status based on provider health
	for provider, healthy := range providerHealth {
		if healthy && provider == summarizer.provider.Name() {
			components["primary"] = string(StatusHealthy)
		} else if healthy {
			components["fallbacks"] = string(StatusHealthy)
		}
	}

	return &HealthReport{
		Status:        status,
		Timestamp:     time.Now(),
		Components:    components,
		Providers:     providerHealth,
		ResponseTimes: responseTimes,
		CacheStats:    cacheStats,
		SuccessRate:   successRate,
		TotalRequests: totalRequests,
		Version:       "1.0.0", // Replace with actual version from your build system
	}, nil
}

// CreateHealthReportJSON generates a JSON health report for the AI summarizer
func CreateHealthReportJSON(summarizer *AISummarizer) (string, error) {
	report, err := CreateHealthReport(summarizer)
	if err != nil {
		return "", err
	}

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal health report: %w", err)
	}

	return string(reportJSON), nil
}

// ResetMetrics resets all metrics for the AI summarizer
func ResetMetrics(summarizer *AISummarizer) error {
	if summarizer == nil {
		return fmt.Errorf("summarizer is nil")
	}

	m := summarizer.GetMetrics()
	if m == nil {
		return fmt.Errorf("metrics collector is nil")
	}

	m.Reset()
	return nil
}
