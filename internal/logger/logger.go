// Package logger provides a structured logging system for the project-memory service.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

// Log level constants
const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
	DISABLED
)

// LogFormat defines how log messages are formatted
type LogFormat int

// Log format constants
const (
	TEXT LogFormat = iota
	JSON
)

var levelNames = map[LogLevel]string{
	DEBUG:    "DEBUG",
	INFO:     "INFO",
	WARN:     "WARN",
	ERROR:    "ERROR",
	FATAL:    "FATAL",
	DISABLED: "DISABLED",
}

// Logger represents a structured logger
type Logger struct {
	level       LogLevel
	format      LogFormat
	out         io.Writer
	fields      map[string]interface{}
	contextPath []string
	mu          sync.Mutex
}

// Config holds configuration options for the logger
type Config struct {
	Level       LogLevel
	Format      LogFormat
	Output      io.Writer
	DefaultTags map[string]interface{}
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:       INFO,
		Format:      TEXT,
		Output:      os.Stderr,
		DefaultTags: map[string]interface{}{"service": "project-memory"},
	}
}

// New creates a new logger with the given configuration
func New(config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	if config.Output == nil {
		config.Output = os.Stderr
	}

	fields := make(map[string]interface{})
	if config.DefaultTags != nil {
		for k, v := range config.DefaultTags {
			fields[k] = v
		}
	}

	return &Logger{
		level:  config.Level,
		format: config.Format,
		out:    config.Output,
		fields: fields,
	}
}

// SetLevel sets the logger's minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetFormat sets the logger's output format
func (l *Logger) SetFormat(format LogFormat) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.format = format
}

// WithField returns a new logger with the field added to its context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create a new fields map with the original fields
	fields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		fields[k] = v
	}

	// Add the new field
	fields[key] = value

	// Return a new logger with the updated fields
	return &Logger{
		level:       l.level,
		format:      l.format,
		out:         l.out,
		fields:      fields,
		contextPath: append([]string{}, l.contextPath...),
	}
}

// WithFields returns a new logger with multiple fields added to its context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create a new fields map with the original fields
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}

	// Add the new fields
	for k, v := range fields {
		newFields[k] = v
	}

	// Return a new logger with the updated fields
	return &Logger{
		level:       l.level,
		format:      l.format,
		out:         l.out,
		fields:      newFields,
		contextPath: append([]string{}, l.contextPath...),
	}
}

// WithContext returns a new logger with a context path
func (l *Logger) WithContext(contexts ...string) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create a new context path with the original and new contexts
	contextPath := append(append([]string{}, l.contextPath...), contexts...)

	// Return a new logger with the updated context path
	return &Logger{
		level:       l.level,
		format:      l.format,
		out:         l.out,
		fields:      l.fields,
		contextPath: contextPath,
	}
}

// Debug logs a message at DEBUG level
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(DEBUG, msg, args...)
}

// Info logs a message at INFO level
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(INFO, msg, args...)
}

// Warn logs a message at WARN level
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(WARN, msg, args...)
}

// Error logs a message at ERROR level
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(ERROR, msg, args...)
}

// Fatal logs a message at FATAL level and then exits with status code 1
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(FATAL, msg, args...)
	os.Exit(1)
}

// DebugContext logs a message at DEBUG level with context
func (l *Logger) DebugContext(ctx string, msg string, args ...interface{}) {
	l.WithContext(ctx).Debug(msg, args...)
}

// InfoContext logs a message at INFO level with context
func (l *Logger) InfoContext(ctx string, msg string, args ...interface{}) {
	l.WithContext(ctx).Info(msg, args...)
}

// WarnContext logs a message at WARN level with context
func (l *Logger) WarnContext(ctx string, msg string, args ...interface{}) {
	l.WithContext(ctx).Warn(msg, args...)
}

// ErrorContext logs a message at ERROR level with context
func (l *Logger) ErrorContext(ctx string, msg string, args ...interface{}) {
	l.WithContext(ctx).Error(msg, args...)
}

// FatalContext logs a message at FATAL level with context and then exits
func (l *Logger) FatalContext(ctx string, msg string, args ...interface{}) {
	l.WithContext(ctx).Fatal(msg, args...)
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Format the message if args are provided
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	// Build the log entry
	timestamp := time.Now().UTC().Format(time.RFC3339)
	levelName := levelNames[level]

	// Add caller information (file and line)
	_, file, line, ok := runtime.Caller(2)
	caller := "unknown"
	if ok {
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Format the log output based on the format type
	var output string
	if l.format == TEXT {
		// Build context path string
		contextStr := ""
		if len(l.contextPath) > 0 {
			contextStr = "[" + strings.Join(l.contextPath, ".") + "] "
		}

		// Build fields string
		fieldsStr := ""
		if len(l.fields) > 0 {
			pairs := make([]string, 0, len(l.fields))
			for k, v := range l.fields {
				pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
			}
			fieldsStr = " " + strings.Join(pairs, " ")
		}

		output = fmt.Sprintf("%s [%s] %s%s (%s)%s\n", timestamp, levelName, contextStr, msg, caller, fieldsStr)
	} else {
		// JSON format
		fieldMap := make(map[string]interface{})
		fieldMap["timestamp"] = timestamp
		fieldMap["level"] = levelName
		fieldMap["message"] = msg
		fieldMap["caller"] = caller

		// Add context path if present
		if len(l.contextPath) > 0 {
			fieldMap["context"] = strings.Join(l.contextPath, ".")
		}

		// Add fields
		for k, v := range l.fields {
			fieldMap[k] = v
		}

		// Convert to JSON (simple implementation, consider using json.Marshal in production)
		pairs := make([]string, 0, len(fieldMap))
		for k, v := range fieldMap {
			var valueStr string
			switch v := v.(type) {
			case string:
				valueStr = fmt.Sprintf("\"%s\"", v)
			default:
				valueStr = fmt.Sprintf("%v", v)
			}
			pairs = append(pairs, fmt.Sprintf("\"%s\":%s", k, valueStr))
		}

		output = fmt.Sprintf("{%s}\n", strings.Join(pairs, ","))
	}

	// Write to output
	fmt.Fprint(l.out, output)
}

// ParseLevel converts a string level to a LogLevel
func ParseLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	case "DISABLED":
		return DISABLED
	default:
		return INFO
	}
}

// Global default logger
var defaultLogger = New(DefaultConfig())

// SetDefaultLogger sets the global default logger
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// GetDefaultLogger returns the global default logger
func GetDefaultLogger() *Logger {
	return defaultLogger
}

// GetLogger returns a logger with the given name as a field
func GetLogger(name string) *Logger {
	return defaultLogger.WithField("name", name)
}

// Debug logs to the default logger at DEBUG level
func Debug(msg string, args ...interface{}) {
	defaultLogger.Debug(msg, args...)
}

// Info logs to the default logger at INFO level
func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

// Warn logs to the default logger at WARN level
func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

// Error logs to the default logger at ERROR level
func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}

// Fatal logs to the default logger at FATAL level and then exits
func Fatal(msg string, args ...interface{}) {
	defaultLogger.Fatal(msg, args...)
}
