package logging

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAutoRotate tests the automatic rotation functionality.
func TestAutoRotate(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "auto-rotate.log")

	// Create writer with auto rotation
	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		t.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}
	defer drw.Close()

	// Write some initial data
	_, err = drw.Write([]byte("initial log message\n"))
	if err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}

	// Verify initial file was created
	today := time.Now().Format("2006-01-02")
	expectedPath := strings.Replace(basePath, ".log", "-"+today+".log", 1)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", expectedPath)
	}

	// Wait a bit to ensure auto rotation goroutine is running
	time.Sleep(100 * time.Millisecond)

	// Write more data
	_, err = drw.Write([]byte("more log data\n"))
	if err != nil {
		t.Fatalf("Failed to write more data: %v", err)
	}

	// Close should stop the auto rotation goroutine
	err = drw.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Verify file still exists and has content
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Log file %s was deleted unexpectedly", expectedPath)
	}
}

// TestAutoRotateWithSlogLogger tests auto rotation with SlogLogger.
func TestAutoRotateWithSlogLogger(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "slog-auto-rotate.log")

	// Create SlogLogger with daily rotation enabled
	logger, err := NewSlogLogger(LevelInfo, "text", logPath, true)
	if err != nil {
		t.Fatalf("Failed to create SlogLogger: %v", err)
	}

	ctx := context.Background()

	// Write some log messages
	logger.Info(ctx, "auto rotation test message 1")
	logger.Info(ctx, "auto rotation test message 2")

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Write more messages
	logger.Warn(ctx, "warning message")
	logger.Error(ctx, "error message")

	// Verify file was created with date suffix
	today := time.Now().Format("2006-01-02")
	expectedPath := strings.Replace(logPath, ".log", "-"+today+".log", 1)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", expectedPath)
	}

	// Read and verify content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "auto rotation test message 1") {
		t.Error("Log file does not contain expected message 1")
	}
	if !strings.Contains(contentStr, "warning message") {
		t.Error("Log file does not contain expected warning message")
	}
	if !strings.Contains(contentStr, "error message") {
		t.Error("Log file does not contain expected error message")
	}
}

// TestAutoRotateConcurrent tests auto rotation with concurrent writes.
func TestAutoRotateConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "concurrent-auto-rotate.log")

	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		t.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}
	defer drw.Close()

	// Concurrent writes while auto rotation is running
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(i int) {
			for j := 0; j < 10; j++ {
				msg := []byte(strings.Repeat("concurrent auto rotate message ", 5) + "\n")
				_, err := drw.Write(msg)
				if err != nil {
					t.Errorf("Concurrent write failed: %v", err)
				}
				time.Sleep(time.Millisecond * 10)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for range 20 {
		<-done
	}

	// Verify file was created
	today := time.Now().Format("2006-01-02")
	expectedPath := strings.Replace(basePath, ".log", "-"+today+".log", 1)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", expectedPath)
	}
}

// TestAutoRotateStop tests that auto rotation stops properly.
func TestAutoRotateStop(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "stop-auto-rotate.log")

	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		t.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}

	// Write some data
	_, err = drw.Write([]byte("before close\n"))
	if err != nil {
		t.Fatalf("Failed to write before close: %v", err)
	}

	// Close should stop auto rotation
	err = drw.Close()
	if err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	// Wait a bit to ensure auto rotation has stopped
	time.Sleep(200 * time.Millisecond)

	// Try to write after close (should fail)
	_, err = drw.Write([]byte("after close\n"))
	if err == nil {
		t.Error("Expected error when writing to closed writer")
	}
}
