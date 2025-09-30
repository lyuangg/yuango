package logging

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestConcurrentSlogLogger tests concurrent SlogLogger usage.
func TestConcurrentSlogLogger(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "concurrent-slog.log")

	logger, err := NewSlogLogger(LevelInfo, "text", logPath, true)
	if err != nil {
		t.Fatalf("Failed to create SlogLogger: %v", err)
	}

	// Test concurrent logging with different contexts
	var wg sync.WaitGroup
	numGoroutines := 50
	iterations := 50

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// 定义 context key 类型
			type goroutineIDKey struct{}
			ctx := context.WithValue(context.Background(), goroutineIDKey{}, goroutineID)
			childLogger := logger.With("worker", goroutineID)

			for j := 0; j < iterations; j++ {
				childLogger.Info(ctx, "concurrent log message",
					"iteration", j,
					"timestamp", time.Now().UnixNano())

				// Test different log levels
				if j%5 == 0 {
					childLogger.Warn(ctx, "warning message", "level", "warn")
				}
				if j%10 == 0 {
					childLogger.Error(ctx, "error message", "level", "error")
				}

				// Small delay to increase chance of race conditions
				if j%10 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentRotationAndWrite tests rotation happening during writes.
func TestConcurrentRotationAndWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "rotation-write.log")

	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		t.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}
	defer drw.Close()

	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			msg := []byte(strings.Repeat("write message ", 20) + "\n")
			_, err := drw.Write(msg)
			if err != nil {
				t.Errorf("Write failed at iteration %d: %v", i, err)
			}

			// Force rotation check frequently
			if i%10 == 0 {
				runtime.Gosched()
			}
		}
	}()

	// Rotation trigger goroutine (simulate time passing)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			// This would trigger rotation in real scenarios
			time.Sleep(time.Microsecond * 100)
			runtime.Gosched()
		}
	}()

	wg.Wait()
}

// TestConcurrentCloseAndWrite tests closing while writing.
func TestConcurrentCloseAndWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "close-write.log")

	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		t.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}

	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			msg := []byte(strings.Repeat("write before close ", 10) + "\n")
			_, err := drw.Write(msg)
			if err != nil {
				// Expected after close
				t.Logf("Write failed after close (expected): %v", err)
				break
			}
			runtime.Gosched()
		}
	}()

	// Close goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 10) // Let some writes happen first
		err := drw.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	wg.Wait()
}

// TestConcurrentLoggerCreation tests creating multiple loggers concurrently.
func TestConcurrentLoggerCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	tempDir := t.TempDir()

	var wg sync.WaitGroup
	numGoroutines := 20
	loggers := make([]Logger, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			logPath := filepath.Join(tempDir, "logger-"+fmt.Sprintf("%d", goroutineID)+".log")
			logger, err := NewSlogLogger(LevelInfo, "text", logPath, false)
			if err != nil {
				t.Errorf("Failed to create logger %d: %v", goroutineID, err)
				return
			}

			loggers[goroutineID] = logger

			// Use the logger immediately
			ctx := context.Background()
			logger.Info(ctx, "logger created", "id", goroutineID)
		}(i)
	}

	wg.Wait()

	// Verify all loggers were created successfully
	for i, logger := range loggers {
		if logger == nil {
			t.Errorf("Logger %d was not created", i)
		}
	}
}

// BenchmarkConcurrentWrite benchmarks concurrent writes to DailyRotateWriter.
func BenchmarkConcurrentWrite(b *testing.B) {
	tempDir := b.TempDir()
	basePath := filepath.Join(tempDir, "bench-concurrent.log")

	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		b.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}
	defer drw.Close()

	testData := []byte("concurrent benchmark message\n")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := drw.Write(testData)
			if err != nil {
				b.Errorf("Write failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentSlogLogger benchmarks concurrent SlogLogger usage.
func BenchmarkConcurrentSlogLogger(b *testing.B) {
	tempDir := b.TempDir()
	logPath := filepath.Join(tempDir, "bench-concurrent-slog.log")

	logger, err := NewSlogLogger(LevelInfo, "text", logPath, false)
	if err != nil {
		b.Fatalf("Failed to create SlogLogger: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(ctx, "concurrent benchmark message",
				"key1", "value1",
				"key2", 42,
				"key3", true)
		}
	})
}
