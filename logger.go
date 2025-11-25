package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	instanceID string
	once       sync.Once
)

// GetInstanceID returns the instance identifier (hostname or pod name)
func GetInstanceID() string {
	once.Do(func() {
		hostname, err := os.Hostname()
		if err != nil {
			instanceID = "unknown"
			return
		}
		instanceID = hostname
		// Check for Kubernetes pod name in environment
		if podName := os.Getenv("POD_NAME"); podName != "" {
			instanceID = podName
		} else if podName := os.Getenv("HOSTNAME"); podName != "" {
			// HOSTNAME is often set to pod name in K8s
			instanceID = podName
		}
	})
	return instanceID
}

// Logger defines the logging interface
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
}

// Field represents a key-value pair in structured logging
type Field struct {
	Key   string
	Value interface{}
}

// F is a helper function to create a Field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Level represents log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelOff
)

// Log level string constants
const (
	levelDebug = "debug"
	levelInfo  = "info"
	levelWarn  = "warn"
	levelError = "error"
	levelOff   = "off"
)

// LoggerConfig configures a logger
type LoggerConfig struct {
	Level      string    // "debug", "info", "warn", "error", "off"
	Format     string    // "json" (default), "text" (for development)
	Writer     io.Writer // Output writer (default: buffered os.Stdout)
	Async      bool      // Use async logging (default: true) - prevents blocking
	BufferSize int       // Buffer size for async logging (default: 1000)
}

// JSONLogger implements structured JSON logging
type JSONLogger struct {
	level   Level
	fields  []Field
	writer  io.Writer
	logChan chan logEntry
	done    chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex // Protects writer for sequential writes
	closed  bool
}

type logEntry struct {
	timestamp time.Time
	level     Level
	message   string
	fields    []Field
}

// NewLogger creates a new logger
// Note: Service name and version will be automatically added from server config
func NewLogger(config *LoggerConfig) Logger {
	if config == nil {
		config = &LoggerConfig{}
	}

	// Set defaults
	level := parseLevel(config.Level)
	// Format is not used in current implementation, but kept for future use
	_ = config.Format

	writer := config.Writer
	if writer == nil {
		writer = bufio.NewWriter(os.Stdout)
	}

	// Buffer size for async logging
	bufferSize := config.BufferSize
	if bufferSize == 0 {
		bufferSize = 1000
	}

	logger := &JSONLogger{
		level:   level,
		fields:  []Field{},
		writer:  writer,
		logChan: make(chan logEntry, bufferSize),
		done:    make(chan struct{}),
	}

	// Always start writer goroutine for async logging
	// The Async flag is kept for API compatibility but always uses async behavior
	// Future: could implement true synchronous logging if needed
	logger.startWriter()

	return logger
}

// parseLevel parses log level string
func parseLevel(level string) Level {
	switch level {
	case levelDebug:
		return LevelDebug
	case levelInfo:
		return LevelInfo
	case levelWarn:
		return LevelWarn
	case levelError:
		return LevelError
	case levelOff:
		return LevelOff
	default:
		return LevelInfo // Default to info
	}
}

// levelString returns string representation of level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return levelDebug
	case LevelInfo:
		return levelInfo
	case LevelWarn:
		return levelWarn
	case LevelError:
		return levelError
	case LevelOff:
		return levelOff
	default:
		return levelInfo
	}
}

// Debug logs a debug message
func (l *JSONLogger) Debug(msg string, fields ...Field) {
	if l.level > LevelDebug {
		return
	}
	l.log(LevelDebug, msg, fields...)
}

// Info logs an info message
func (l *JSONLogger) Info(msg string, fields ...Field) {
	if l.level > LevelInfo {
		return
	}
	l.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message
func (l *JSONLogger) Warn(msg string, fields ...Field) {
	if l.level > LevelWarn {
		return
	}
	l.log(LevelWarn, msg, fields...)
}

// Error logs an error message
func (l *JSONLogger) Error(msg string, fields ...Field) {
	if l.level > LevelError {
		return
	}
	l.log(LevelError, msg, fields...)
}

// log queues a log entry (non-blocking)
func (l *JSONLogger) log(level Level, msg string, fields ...Field) {
	if l.closed {
		return
	}

	entry := logEntry{
		timestamp: time.Now().UTC(),
		level:     level,
		message:   msg,
		fields:    append(l.fields, fields...),
	}

	// Try to queue log entry (non-blocking)
	select {
	case l.logChan <- entry:
		// Successfully queued
	default:
		// Channel full - drop log to prevent blocking
		// Optionally could log to stderr, but we'll just drop it
	}
}

// startWriter starts the background goroutine that writes logs
func (l *JSONLogger) startWriter() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		for entry := range l.logChan {
			l.writeLog(entry)
		}
	}()
}

// writeLog writes a single log entry (called sequentially)
func (l *JSONLogger) writeLog(entry logEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Build log map
	logMap := make(map[string]interface{})
	logMap["timestamp"] = entry.timestamp.Format(time.RFC3339Nano)
	logMap["level"] = entry.level.String()
	logMap["message"] = entry.message

	// Add fields
	for _, field := range entry.fields {
		logMap[field.Key] = field.Value
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(logMap)
	if err != nil {
		// Fallback to simple message if JSON marshaling fails
		timestamp := time.Now().UTC().Format(time.RFC3339Nano)
		errorMsg := "Failed to marshal log: " + err.Error()
		jsonBytes = []byte(fmt.Sprintf(`{"timestamp":%q,"level":"error","message":%q}`, timestamp, errorMsg))
	}

	// Write log
	jsonBytes = append(jsonBytes, '\n')
	if _, writeErr := l.writer.Write(jsonBytes); writeErr != nil {
		// Can't log the error (would cause infinite loop), but we tried
		_ = writeErr
	}

	// Flush if buffered writer
	if bw, ok := l.writer.(*bufio.Writer); ok {
		_ = bw.Flush()
	}
}

// WithFields returns a new logger with additional fields
func (l *JSONLogger) WithFields(fields ...Field) Logger {
	// Create new fields slice to avoid modifying original
	newFields := make([]Field, len(l.fields), len(l.fields)+len(fields))
	copy(newFields, l.fields)
	newFields = append(newFields, fields...)

	// Create new logger without copying locks (wg, mu)
	// New logger shares the same logChan, so it will work with the same writer goroutine
	return &JSONLogger{
		level:   l.level,
		fields:  newFields,
		writer:  l.writer,
		logChan: l.logChan,
		done:    l.done,
		closed:  l.closed,
		// Note: wg and mu are not copied to avoid lock copying issues
		// They will be zero values, but that's OK since we share logChan
	}
}

// WithContext returns a new logger with context fields
func (l *JSONLogger) WithContext(_ context.Context) Logger {
	// Extract context fields if available
	// For now, just return logger with existing fields
	// Can be extended to extract trace IDs, etc.
	return l
}

// Close closes the logger and flushes pending logs
func (l *JSONLogger) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	close(l.logChan)
	l.wg.Wait() // Wait for all logs to be written

	// Flush writer
	if bw, ok := l.writer.(*bufio.Writer); ok {
		return bw.Flush()
	}
	return nil
}
