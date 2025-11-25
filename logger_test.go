package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config *LoggerConfig
		want   Level
	}{
		{
			name:   "Default config",
			config: nil,
			want:   LevelInfo,
		},
		{
			name: "Debug level",
			config: &LoggerConfig{
				Level: "debug",
			},
			want: LevelDebug,
		},
		{
			name: "Info level",
			config: &LoggerConfig{
				Level: "info",
			},
			want: LevelInfo,
		},
		{
			name: "Warn level",
			config: &LoggerConfig{
				Level: "warn",
			},
			want: LevelWarn,
		},
		{
			name: "Error level",
			config: &LoggerConfig{
				Level: "error",
			},
			want: LevelError,
		},
		{
			name: "Off level",
			config: &LoggerConfig{
				Level: "off",
			},
			want: LevelOff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.config)
			jsonLogger, ok := logger.(*JSONLogger)
			if !ok {
				t.Fatalf("Expected *JSONLogger, got %T", logger)
			}
			if jsonLogger.level != tt.want {
				t.Errorf("Expected level %v, got %v", tt.want, jsonLogger.level)
			}
		})
	}
}

func TestLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  "debug",
		Format: "json",
		Writer: &buf,
		Async:  false, // Synchronous for testing
	}).(*JSONLogger)

	// Test all log levels
	logger.Debug("debug message", F("key1", "value1"))
	logger.Info("info message", F("key2", "value2"))
	logger.Warn("warn message", F("key3", "value3"))
	logger.Error("error message", F("key4", "value4"))

	// Close logger to flush
	_ = logger.Close()

	// Verify logs were written
	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("Debug message not found")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info message not found")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message not found")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message not found")
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  "warn",
		Format: "json",
		Writer: &buf,
		Async:  false,
	}).(*JSONLogger)

	// These should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")

	// These should be logged
	logger.Warn("warn message")
	logger.Error("error message")

	_ = logger.Close()

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should be logged")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should be logged")
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  "info",
		Format: "json",
		Writer: &buf,
		Async:  false,
	})

	logger.Info("test message", F("key1", "value1"), F("key2", 123))

	jsonLogger := logger.(*JSONLogger)
	_ = jsonLogger.Close()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		t.Fatal("No log output")
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if logEntry["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", logEntry["message"])
	}
	if logEntry["level"] != "info" {
		t.Errorf("Expected level 'info', got %v", logEntry["level"])
	}
	if logEntry["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got %v", logEntry["key1"])
	}
	if logEntry["key2"] != float64(123) { // JSON numbers are float64
		t.Errorf("Expected key2=123, got %v", logEntry["key2"])
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:  "info",
		Writer: &buf,
		Async:  false, // Note: Async is always true internally, but test works with it
	})

	loggerWithFields := logger.WithFields(
		F("service", "test-service"),
		F("version", "1.0.0"),
	)

	loggerWithFields.Info("test message", F("request_id", "req-123"))

	// Close the original logger to flush logs (they share the same channel)
	jsonLogger := logger.(*JSONLogger)
	_ = jsonLogger.Close()

	// Give async logger time to write (if needed)
	time.Sleep(50 * time.Millisecond)

	output := buf.String()
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatalf("No log output received. Buffer: %q", output)
	}
	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON: %v. Output: %q", err, lines[0])
	}

	if logEntry["service"] != "test-service" {
		t.Errorf("Expected service='test-service', got %v", logEntry["service"])
	}
	if logEntry["version"] != "1.0.0" {
		t.Errorf("Expected version='1.0.0', got %v", logEntry["version"])
	}
	if logEntry["request_id"] != "req-123" {
		t.Errorf("Expected request_id='req-123', got %v", logEntry["request_id"])
	}
}

func TestLogger_AsyncLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:      "info",
		Writer:     &buf,
		Async:      true,
		BufferSize: 100,
	})

	// Log multiple messages
	for i := 0; i < 10; i++ {
		logger.Info("test message", F("index", i))
	}

	// Give async logger time to process
	time.Sleep(100 * time.Millisecond)

	jsonLogger := logger.(*JSONLogger)
	_ = jsonLogger.Close()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 10 {
		t.Errorf("Expected 10 log lines, got %d", len(lines))
	}
}

func TestLogger_OrderMaintenance(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:      "info",
		Writer:     &buf,
		Async:      true,
		BufferSize: 100,
	})

	// Log messages in order
	for i := 0; i < 5; i++ {
		logger.Info("message", F("order", i))
	}

	jsonLogger := logger.(*JSONLogger)
	_ = jsonLogger.Close()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 5 {
		t.Fatalf("Expected 5 log lines, got %d", len(lines))
	}

	// Verify order (check timestamps or order field)
	for i, line := range lines {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}
		if logEntry["order"] != float64(i) {
			t.Errorf("Expected order %d, got %v", i, logEntry["order"])
		}
	}
}

func TestGetInstanceID(t *testing.T) {
	id1 := GetInstanceID()
	id2 := GetInstanceID()

	// Should return the same ID (cached)
	if id1 != id2 {
		t.Errorf("Expected same instance ID, got %s and %s", id1, id2)
	}

	// Should not be empty
	if id1 == "" {
		t.Error("Instance ID should not be empty")
	}
}

func TestLogger_Close(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:      "info",
		Writer:     &buf,
		Async:      true,
		BufferSize: 100,
	})

	logger.Info("test message")

	jsonLogger := logger.(*JSONLogger)
	if err := jsonLogger.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Close again should be safe
	if err := jsonLogger.Close(); err != nil {
		t.Errorf("Second Close() returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Log message not found after close")
	}
}

func TestLogger_WithContext(t *testing.T) {
	logger := NewLogger(&LoggerConfig{
		Level: "info",
	})

	ctx := context.Background()
	loggerWithContext := logger.WithContext(ctx)

	// Should return a logger (implementation may vary)
	if loggerWithContext == nil {
		t.Error("WithContext should return a logger")
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarn, "warn"},
		{LevelError, "error"},
		{LevelOff, "off"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger_DefaultWriter(t *testing.T) {
	// Test that default writer is set to stdout
	logger := NewLogger(&LoggerConfig{
		Level: "info",
	})

	jsonLogger := logger.(*JSONLogger)
	if jsonLogger.writer == nil {
		t.Error("Default writer should not be nil")
	}
}

func TestLogger_BufferFull(t *testing.T) {
	// Test that logger doesn't block when buffer is full
	var buf bytes.Buffer
	logger := NewLogger(&LoggerConfig{
		Level:      "info",
		Writer:     &buf,
		Async:      true,
		BufferSize: 2, // Small buffer
	})

	// Fill buffer and then some
	for i := 0; i < 10; i++ {
		logger.Info("message", F("index", i))
	}

	// Should not block
	jsonLogger := logger.(*JSONLogger)
	_ = jsonLogger.Close()

	// Some logs may be dropped, but that's expected
	output := buf.String()
	if output == "" {
		t.Error("At least some logs should be written")
	}
}
