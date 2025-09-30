package logging

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewFromConfig_SpecialCases tests special cases for NewFromConfig
func TestNewFromConfig_SpecialCases(t *testing.T) {
	type Config struct {
		Level  string
		Format string
		Output string
		Daily  bool
	}

	testCases := []struct {
		name        string
		config      interface{}
		shouldError bool
	}{
		{
			name: "config with pointer fields",
			config: &struct {
				Level  *string
				Format *string
				Output *string
				Daily  *bool
			}{},
			shouldError: false,
		},
		{
			name: "config with interface fields",
			config: struct {
				Level  interface{}
				Format interface{}
				Output interface{}
				Daily  interface{}
			}{
				Level:  "debug",
				Format: "json",
				Output: "stdout",
				Daily:  true,
			},
			shouldError: false,
		},
		{
			name: "config with unexported fields",
			config: struct {
				level  string
				format string
				output string
				daily  bool
			}{},
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger, err := NewFromConfig(tc.config)
			if tc.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tc.shouldError && logger == nil {
				t.Error("Expected non-nil logger")
			}
		})
	}
}

// TestDailyRotateWriter_WriteAndClose tests write operations and close behavior
func TestDailyRotateWriter_WriteAndClose(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "write_close_test.log")

	writer, err := NewDailyRotateWriter(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Test multiple writes
	for i := range 5 {
		_, err := writer.Write([]byte("test\n"))
		if err != nil {
			t.Errorf("Write failed on iteration %d: %v", i, err)
		}
	}

	// Test concurrent writes
	done := make(chan bool)
	for i := range 10 {
		go func(i int) {
			_, err := writer.Write([]byte("concurrent test\n"))
			if err != nil {
				t.Errorf("Concurrent write failed on iteration %d: %v", i, err)
			}
			done <- true
		}(i)
	}

	// Wait for all writes to complete
	for range 10 {
		<-done
	}

	// Test close
	if err := writer.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestAutoRotate_TimeoutAndStop tests auto-rotation with timeout and stop conditions
func TestAutoRotate_TimeoutAndStop(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "auto_rotate_test.log")

	writer, err := NewDailyRotateWriter(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Write initial data
	_, err = writer.Write([]byte("initial data\n"))
	if err != nil {
		t.Fatal(err)
	}

	// Let the auto-rotate goroutine run for a bit
	time.Sleep(2 * time.Second)

	// Write more data
	_, err = writer.Write([]byte("more data\n"))
	if err != nil {
		t.Fatal(err)
	}

	// Close the writer (should stop auto-rotate)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	// 验证文件是否存在
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".log" {
			found = true
			break
		}
	}

	if !found {
		t.Error("No log file was created")
	}
}

// TestWithContext_EdgeCases tests edge cases for WithContext
func TestWithContext_EdgeCases(t *testing.T) {
	logger, err := NewSlogLogger(LevelDebug, "text", "stdout", false)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name    string
		ctx     context.Context
		logFunc func(logger Logger)
	}{
		{
			name: "background context",
			ctx:  context.Background(),
			logFunc: func(logger Logger) {
				logger.Debug(context.Background(), "debug message")
				logger.Info(context.Background(), "info message")
				logger.Warn(context.Background(), "warn message")
				logger.Error(context.Background(), "error message")
			},
		},
		{
			name: "background context",
			ctx:  context.Background(),
			logFunc: func(logger Logger) {
				logger.Debug(context.Background(), "debug message")
				logger.Info(context.Background(), "info message")
				logger.Warn(context.Background(), "warn message")
				logger.Error(context.Background(), "error message")
			},
		},
		{
			name: "context with values",
			ctx: func() context.Context {
				// Define custom type for context key
				type ctxKey struct{}
				return context.WithValue(context.Background(), ctxKey{}, "value")
			}(),
			logFunc: func(logger Logger) {
				logger.Info(context.Background(), "info message")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contextLogger := logger.WithContext(tc.ctx)
			tc.logFunc(contextLogger)
		})
	}
}
