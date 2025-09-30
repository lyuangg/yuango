package logging

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSlogLoggerAllLevels tests all log levels and their filtering.
func TestSlogLoggerAllLevels(t *testing.T) {
	tempDir := t.TempDir()

	levels := []struct {
		name     string
		logLevel Level
		levelStr string
	}{
		{"debug", LevelDebug, "debug"},
		{"info", LevelInfo, "info"},
		{"warn", LevelWarn, "warn"},
		{"error", LevelError, "error"},
	}

	for _, tc := range levels {
		t.Run(tc.name, func(t *testing.T) {
			logPath := filepath.Join(tempDir, tc.levelStr+".log")

			logger, err := NewSlogLogger(tc.logLevel, "text", logPath, false)
			if err != nil {
				t.Fatalf("Failed to create logger for %s: %v", tc.name, err)
			}

			ctx := context.Background()

			// Test all log methods
			logger.Debug(ctx, "debug message", "level", "debug")
			logger.Info(ctx, "info message", "level", "info")
			logger.Warn(ctx, "warn message", "level", "warn")
			logger.Error(ctx, "error message", "level", "error")

			// Test Enabled method for each level
			if !logger.Enabled(ctx, tc.logLevel) {
				t.Errorf("Level %s should be enabled", tc.name)
			}

			// Test higher levels
			if tc.logLevel > LevelDebug && logger.Enabled(ctx, LevelDebug) {
				t.Errorf("Debug should be disabled when level is %s", tc.name)
			}

			if tc.logLevel > LevelInfo && logger.Enabled(ctx, LevelInfo) {
				t.Errorf("Info should be disabled when level is %s", tc.name)
			}
		})
	}
}

// TestSlogLogger_ContextBinding tests context binding functionality.
func TestSlogLogger_ContextBinding(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "context.log")

	logger, err := NewSlogLogger(LevelDebug, "text", logPath, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Define custom key types
	type requestIDKey struct{}
	type userIDKey struct{}

	// Create context with values using typed keys
	ctx := context.WithValue(context.Background(), requestIDKey{}, "req-123")
	ctx = context.WithValue(ctx, userIDKey{}, "user-456")

	// Test context binding
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info(context.TODO(), "message with bound context")

	// Test With method chaining
	chainedLogger := logger.With("service", "test-service").
		WithContext(ctx).
		With("version", "v1.0.0")

	chainedLogger.Debug(context.TODO(), "chained logger message",
		"additional_field", "value")

	// Test multiple context usages
	logger.With("component", "handler").WithContext(ctx).Info(context.TODO(), "handler message")
	logger.WithContext(ctx).With("component", "service").Warn(context.TODO(), "service message")

	// Verify file content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	expectedFields := []string{
		"service=test-service",
		"version=v1.0.0",
		"component=handler",
		"component=service",
	}

	for _, field := range expectedFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("Expected field %s not found in log", field)
		}
	}
}

// TestDailyRotateWriter_EdgeCases tests edge cases for DailyRotateWriter.
func TestDailyRotateWriter_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("empty file path", func(t *testing.T) {
		// Empty path should create a file in current directory
		_, err := NewDailyRotateWriter("")
		if err != nil {
			t.Logf("Empty file path creates error (expected): %v", err)
			// This is acceptable behavior
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent", "test.log")
		drw, err := NewDailyRotateWriter(nonExistentPath)
		if err != nil {
			t.Fatalf("Failed to create writer: %v", err)
		}
		defer drw.Close()

		// Write should create directory and file
		_, err = drw.Write([]byte("test message\n"))
		if err != nil {
			t.Errorf("Failed to write to non-existent directory: %v", err)
		}

		// Verify directory was created
		if _, err := os.Stat(filepath.Dir(nonExistentPath)); os.IsNotExist(err) {
			t.Error("Directory should have been created")
		}
	})

	t.Run("concurrent writes large data", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping in short mode")
		}

		basePath := filepath.Join(tempDir, "large.log")
		drw, err := NewDailyRotateWriter(basePath)
		if err != nil {
			t.Fatalf("Failed to create writer: %v", err)
		}
		defer drw.Close()

		// Write large amount of data concurrently
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(i int) {
				for j := 0; j < 100; j++ {
					largeMessage := []byte(strings.Repeat("large message content ", 1000) + "\n")
					_, err := drw.Write(largeMessage)
					if err != nil {
						t.Errorf("Write failed: %v", err)
					}
				}
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("write after close", func(t *testing.T) {
		basePath := filepath.Join(tempDir, "closed.log")
		drw, err := NewDailyRotateWriter(basePath)
		if err != nil {
			t.Fatalf("Failed to create writer: %v", err)
		}

		// Write before close
		_, err = drw.Write([]byte("before close\n"))
		if err != nil {
			t.Errorf("Write failed before close: %v", err)
		}

		// Close
		err = drw.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}

		// Write after close should fail
		_, err = drw.Write([]byte("after close\n"))
		if err == nil {
			t.Error("Expected error when writing to closed writer")
		}
	})

	t.Run("multiple close calls", func(t *testing.T) {
		panic := make(chan bool, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					panic <- true
				} else {
					panic <- false
				}
			}()

			basePath := filepath.Join(tempDir, "multiple-close.log")
			drw, err := NewDailyRotateWriter(basePath)
			if err != nil {
				t.Errorf("Failed to create writer: %v", err)
				panic <- false
				return
			}

			// Multiple close calls should not panic
			drw.Close()
			drw.Close()
			drw.Close()
		}()

		if didPanic := <-panic; didPanic {
			t.Error("Multiple close calls should not panic")
		}
	})
}

// TestToSlogComprehensive tests the toSlog conversion comprehensively.
func TestToSlogComprehensive(t *testing.T) {
	testCases := []struct {
		name   string
		level  Level
		expect slog.Level // Expected slog level value
	}{
		{"debug", LevelDebug, slog.LevelDebug},
		{"info", LevelInfo, slog.LevelInfo},
		{"warn", LevelWarn, slog.LevelWarn},
		{"error", LevelError, slog.LevelError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			slogLevel := tc.level.toSlog()
			if slogLevel != tc.expect {
				t.Errorf("toSlog() = %d, want %d", slogLevel, tc.expect)
			}
		})
	}
}

// TestSlogLogger_PerformanceLevels tests performance across different levels.
func TestSlogLogger_PerformanceLevels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	tempDir := t.TempDir()

	testCases := []struct {
		name  string
		level Level
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"error", LevelError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logPath := filepath.Join(tempDir, tc.name+".perf.log")

			logger, err := NewSlogLogger(tc.level, "json", logPath, false)
			if err != nil {
				t.Fatalf("Failed to create %s logger: %v", tc.name, err)
			}

			ctx := context.Background()

			// Performance test
			start := time.Now()
			for i := 0; i < 1000; i++ {
				logger.Debug(ctx, "debug message", "iteration", i)
				logger.Info(ctx, "info message", "iteration", i)
				logger.Warn(ctx, "warn message", "iteration", i)
				logger.Error(ctx, "error message", "iteration", i)
			}
			duration := time.Since(start)

			t.Logf("Level %s: 4000 messages in %v", tc.name, duration)
		})
	}
}
