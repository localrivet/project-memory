// Package telemetry provides metrics collection and reporting
// for monitoring the LLM-Memory service performance.
package telemetry

import (
	"fmt"
	"sync"
	"time"
)

// MetricsCollector provides a thread-safe interface for collecting
// application metrics for monitoring and troubleshooting.
type MetricsCollector struct {
	counters   map[string]int64
	gauges     map[string]float64
	timers     map[string][]time.Duration
	latestTime map[string]time.Time
	mu         sync.RWMutex
}

// SummarizerMetrics defines constants for metrics related to the AI Summarizer
const (
	// API Call counts by provider
	MetricAPICallsAnthropic = "summarizer.api_calls.anthropic"
	MetricAPICallsOpenAI    = "summarizer.api_calls.openai"
	MetricAPICallsGoogle    = "summarizer.api_calls.google"
	MetricAPICallsXAI       = "summarizer.api_calls.xai"

	// Success/failure metrics
	MetricAPICallsSuccess = "summarizer.api_calls.success"
	MetricAPICallsFailure = "summarizer.api_calls.failure"

	// Retry metrics
	MetricRetryAttempts = "summarizer.retry_attempts"
	MetricRetrySuccess  = "summarizer.retry_success"

	// Fallback metrics
	MetricFallbackAttempts = "summarizer.fallback_attempts"
	MetricFallbackSuccess  = "summarizer.fallback_success"

	// Cache metrics
	MetricCacheHits   = "summarizer.cache.hits"
	MetricCacheMisses = "summarizer.cache.misses"
	MetricCacheSize   = "summarizer.cache.size"

	// Response times
	MetricResponseTimeAnthropic = "summarizer.response_time.anthropic"
	MetricResponseTimeOpenAI    = "summarizer.response_time.openai"
	MetricResponseTimeGoogle    = "summarizer.response_time.google"
	MetricResponseTimeXAI       = "summarizer.response_time.xai"

	// Provider health
	MetricProviderHealthAnthropic = "summarizer.health.anthropic"
	MetricProviderHealthOpenAI    = "summarizer.health.openai"
	MetricProviderHealthGoogle    = "summarizer.health.google"
	MetricProviderHealthXAI       = "summarizer.health.xai"
)

// NewMetricsCollector creates a new MetricsCollector instance
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		timers:     make(map[string][]time.Duration),
		latestTime: make(map[string]time.Time),
	}
}

// IncrementCounter increments a named counter by the specified amount
func (m *MetricsCollector) IncrementCounter(name string, amount int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name] += amount
}

// SetGauge sets a named gauge to the specified value
func (m *MetricsCollector) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges[name] = value
}

// RecordTimer records a duration for the specified timer
func (m *MetricsCollector) RecordTimer(name string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.timers[name]; !exists {
		m.timers[name] = make([]time.Duration, 0)
	}

	m.timers[name] = append(m.timers[name], duration)

	// Limit the number of stored durations to avoid unbounded growth
	if len(m.timers[name]) > 100 {
		m.timers[name] = m.timers[name][1:]
	}
}

// RecordTimestamp records the current time for the specified event
func (m *MetricsCollector) RecordTimestamp(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latestTime[name] = time.Now()
}

// GetCounter retrieves the current value of a counter
func (m *MetricsCollector) GetCounter(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.counters[name]
}

// GetGauge retrieves the current value of a gauge
func (m *MetricsCollector) GetGauge(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.gauges[name]
}

// GetTimerAverage calculates the average duration for a timer
func (m *MetricsCollector) GetTimerAverage(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	durations, exists := m.timers[name]
	if !exists || len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

// GetTimerP95 calculates the 95th percentile duration for a timer
func (m *MetricsCollector) GetTimerP95(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	durations, exists := m.timers[name]
	if !exists || len(durations) == 0 {
		return 0
	}

	// Sort durations (simple bubble sort for now)
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// Calculate p95 index
	idx := int(float64(len(sorted)) * 0.95)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

// GetTimeSince calculates the time elapsed since a recorded timestamp
func (m *MetricsCollector) GetTimeSince(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	timestamp, exists := m.latestTime[name]
	if !exists {
		return 0
	}

	return time.Since(timestamp)
}

// GetReport generates a report of all collected metrics
func (m *MetricsCollector) GetReport() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := "Metrics Report:\n"
	report += "==============\n\n"

	report += "Counters:\n"
	for name, value := range m.counters {
		report += fmt.Sprintf("  %s: %d\n", name, value)
	}

	report += "\nGauges:\n"
	for name, value := range m.gauges {
		report += fmt.Sprintf("  %s: %.2f\n", name, value)
	}

	report += "\nTimers (avg):\n"
	for name := range m.timers {
		avg := m.GetTimerAverage(name)
		p95 := m.GetTimerP95(name)
		report += fmt.Sprintf("  %s: avg=%v p95=%v count=%d\n",
			name, avg, p95, len(m.timers[name]))
	}

	report += "\nTime Since:\n"
	for name, timestamp := range m.latestTime {
		report += fmt.Sprintf("  %s: %v ago (%s)\n",
			name, time.Since(timestamp), timestamp.Format(time.RFC3339))
	}

	return report
}

// Reset clears all collected metrics
func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]int64)
	m.gauges = make(map[string]float64)
	m.timers = make(map[string][]time.Duration)
	m.latestTime = make(map[string]time.Time)
}
