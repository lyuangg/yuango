package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDailyRotateWriter tests the daily rotation functionality.
func TestDailyRotateWriter(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "test.log")

	// Test 1: Create writer and write data
	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		t.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}
	defer drw.Close()

	// Test basic write functionality
	testData := []byte("test log message\n")
	n, err := drw.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Verify file creation with correct date suffix
	today := time.Now().Format("2006-01-02")
	expectedPath := strings.Replace(basePath, ".log", "-"+today+".log", 1)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", expectedPath)
	}

	// Test multiple writes
	_, err = drw.Write([]byte("second message\n"))
	if err != nil {
		t.Fatalf("Failed to write second message: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify both messages are present
	if !strings.Contains(string(content), "test log message") ||
		!strings.Contains(string(content), "second message") {
		t.Error("Log file does not contain all expected messages")
	}
}

// TestDailyRotateWriter_Close tests the Close method.
func TestDailyRotateWriter_Close(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "test.log")

	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		t.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}

	// Write some data
	_, err = drw.Write([]byte("test data\n"))
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	// Close the writer
	err = drw.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Try to write after close (should fail gracefully)
	_, err = drw.Write([]byte("after close\n"))
	if err == nil {
		t.Error("Expected error when writing to closed writer")
	}
}

// TestDailyRotateWriter_ConcurrentRotation tests concurrent rotation scenarios.
func TestDailyRotateWriter_ConcurrentRotation(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "concurrent.log")

	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		t.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}
	defer drw.Close()

	// 设置并发参数
	const (
		goroutines           = 100 // 并发写入的 goroutine 数量
		messagesPerGoroutine = 100 // 每个 goroutine 写入的消息数量
	)

	// 创建同步机制
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)
	writeCount := int32(0)

	// 启动并发写入
	for i := range goroutines {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := range messagesPerGoroutine {
				// 生成唯一的消息
				msg := fmt.Sprintf("routine-%d-msg-%d\n", routineID, j)

				// 写入消息
				_, err := drw.Write([]byte(msg))
				if err != nil {
					errCh <- fmt.Errorf("routine %d write failed: %v", routineID, err)
					return
				}
				atomic.AddInt32(&writeCount, 1)
			}
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 检查是否有错误
	select {
	case err := <-errCh:
		t.Errorf("Concurrent write error: %v", err)
		return
	default:
		// 没有错误，继续
	}

	// 验证所有消息都被写入
	expectedMessages := goroutines * messagesPerGoroutine
	actualMessages := int(atomic.LoadInt32(&writeCount))
	if actualMessages != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, actualMessages)
	}

	// 验证文件内容
	today := time.Now().Format("2006-01-02")
	expectedPath := strings.Replace(basePath, ".log", "-"+today+".log", 1)
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// 检查文件行数
	lines := strings.Split(string(content), "\n")
	if len(lines)-1 != expectedMessages { // -1 是因为最后一个换行符会产生一个空行
		t.Errorf("Expected %d lines in log file, got %d", expectedMessages, len(lines)-1)
	}
}

// TestSlogLogger tests the SlogLogger functionality.
func TestSlogLogger(t *testing.T) {
	// Test with stdout
	logger, err := NewSlogLogger(LevelInfo, "text", "stdout", false)
	if err != nil {
		t.Fatalf("Failed to create SlogLogger: %v", err)
	}

	ctx := context.Background()

	// Test all log levels
	logger.Debug(ctx, "debug message", "key", "value")
	logger.Info(ctx, "info message", "key", "value")
	logger.Warn(ctx, "warn message", "key", "value")
	logger.Error(ctx, "error message", "key", "value")

	// Test With method
	childLogger := logger.With("service", "test")
	childLogger.Info(ctx, "child logger message")

	// Test WithContext method
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info(context.TODO(), "context logger message")

	// Test Enabled method
	if !logger.Enabled(ctx, LevelInfo) {
		t.Error("Expected LevelInfo to be enabled")
	}
	if logger.Enabled(ctx, LevelDebug) {
		t.Error("Expected LevelDebug to be disabled at Info level")
	}
}

// TestSlogLogger_JSONFormat tests JSON format output.
func TestSlogLogger_JSONFormat(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.json")

	logger, err := NewSlogLogger(LevelInfo, "json", logPath, false)
	if err != nil {
		t.Fatalf("Failed to create JSON SlogLogger: %v", err)
	}

	ctx := context.Background()
	logger.Info(ctx, "json test message", "key1", "value1", "key2", 42)

	// Verify file was created and contains JSON
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Expected log file %s was not created", logPath)
	}

	// Read and verify JSON content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// 验证是否是有效的JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Errorf("Invalid JSON format: %v", err)
	}

	// 验证JSON结构和字段值
	expectedFields := map[string]interface{}{
		"msg":   "json test message",
		"key1":  "value1",
		"key2":  float64(42), // JSON numbers are decoded as float64
		"level": "INFO",
	}

	for field, expectedValue := range expectedFields {
		value, exists := logEntry[field]
		if !exists {
			t.Errorf("Missing field '%s' in JSON output", field)
			continue
		}
		if !reflect.DeepEqual(value, expectedValue) {
			t.Errorf("Field '%s' has wrong value. Expected: %v, Got: %v", field, expectedValue, value)
		}
	}

	// 验证时间戳字段
	if _, exists := logEntry["time"]; !exists {
		t.Error("Missing 'time' field in JSON output")
	}
}

// TestSlogLogger_DailyRotation tests daily rotation integration with SlogLogger.
func TestSlogLogger_DailyRotation(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "daily.log")

	// 创建带有日志轮转的 logger
	logger, err := NewSlogLogger(LevelInfo, "json", logPath, true, 3) // 使用 JSON 格式并限制保留 3 个文件
	if err != nil {
		t.Fatalf("Failed to create daily rotation SlogLogger: %v", err)
	}

	ctx := context.Background()

	// 测试不同日志级别
	logger.Debug(ctx, "debug message", "visible", false) // 不应该被记录
	logger.Info(ctx, "info message", "test", true)
	logger.Warn(ctx, "warning message", "code", 100)
	logger.Error(ctx, "error message", "fatal", false)

	// 验证文件创建和格式
	today := time.Now().Format("2006-01-02")
	expectedPath := strings.Replace(logPath, ".log", "-"+today+".log", 1)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected daily log file %s was not created", expectedPath)
		return
	}

	// 读取并验证日志内容
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// 验证日志格式和内容
	if !strings.Contains(logContent, `"level":"INFO"`) ||
		!strings.Contains(logContent, `"msg":"info message"`) ||
		!strings.Contains(logContent, `"test":true`) {
		t.Error("Log file doesn't contain expected JSON format or content")
	}

	// 验证日志级别过滤
	if strings.Contains(logContent, "debug message") {
		t.Error("Debug message should not be logged at Info level")
	}

	// 验证所有应该记录的消息都存在
	expectedMessages := []string{"info message", "warning message", "error message"}
	for _, msg := range expectedMessages {
		if !strings.Contains(logContent, msg) {
			t.Errorf("Expected message '%s' not found in log file", msg)
		}
	}
}

// TestSlogLogger_Levels tests different log levels.
func TestSlogLogger_Levels(t *testing.T) {
	tempDir := t.TempDir()

	// 测试 Debug 级别
	debugLogPath := filepath.Join(tempDir, "debug.log")
	debugLogger, err := NewSlogLogger(LevelDebug, "text", debugLogPath, false)
	if err != nil {
		t.Fatalf("Failed to create debug logger: %v", err)
	}

	ctx := context.Background()
	debugLogger.Debug(ctx, "debug message")
	debugLogger.Info(ctx, "info message")
	debugLogger.Warn(ctx, "warn message")
	debugLogger.Error(ctx, "error message")

	// 验证 Debug 级别的日志文件
	content, err := os.ReadFile(debugLogPath)
	if err != nil {
		t.Fatalf("Failed to read debug log file: %v", err)
	}
	debugContent := string(content)

	// Debug 级别应该记录所有消息
	expectedMessages := []string{"debug message", "info message", "warn message", "error message"}
	for _, msg := range expectedMessages {
		if !strings.Contains(debugContent, msg) {
			t.Errorf("Debug log should contain '%s'", msg)
		}
	}

	// 测试 Error 级别
	errorLogPath := filepath.Join(tempDir, "error.log")
	errorLogger, err := NewSlogLogger(LevelError, "text", errorLogPath, false)
	if err != nil {
		t.Fatalf("Failed to create error logger: %v", err)
	}

	errorLogger.Debug(ctx, "debug message") // 应该被过滤
	errorLogger.Info(ctx, "info message")   // 应该被过滤
	errorLogger.Warn(ctx, "warn message")   // 应该被过滤
	errorLogger.Error(ctx, "error message") // 应该被记录

	// 验证 Error 级别的日志文件
	content, err = os.ReadFile(errorLogPath)
	if err != nil {
		t.Fatalf("Failed to read error log file: %v", err)
	}
	errorContent := string(content)

	// Error 级别不应该包含低级别的消息
	filteredMessages := []string{"debug message", "info message", "warn message"}
	for _, msg := range filteredMessages {
		if strings.Contains(errorContent, msg) {
			t.Errorf("Error log should not contain '%s'", msg)
		}
	}

	// Error 级别应该包含 error 消息
	if !strings.Contains(errorContent, "error message") {
		t.Error("Error log should contain 'error message'")
	}
}

// TestSlogLogger_WithAndWithContext tests the With and WithContext methods.
func TestSlogLogger_WithAndWithContext(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "with.log")

	logger, err := NewSlogLogger(LevelInfo, "text", logPath, false)
	if err != nil {
		t.Fatalf("Failed to create SlogLogger: %v", err)
	}

	ctx := context.Background()
	// 在上下文中添加值
	type ctxKey string
	ctx = context.WithValue(ctx, ctxKey("request_id"), "123456")

	// 测试 With 方法
	serviceLogger := logger.With("service", "test-service", "version", "1.0.0")
	serviceLogger.Info(ctx, "service message")

	// 测试 WithContext 方法
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info(context.TODO(), "context message") // 使用 TODO() 验证绑定的上下文是否生效

	// 测试链式调用
	chainedLogger := logger.With("component", "handler").WithContext(ctx)
	chainedLogger.Info(context.TODO(), "chained message")

	// 读取并验证日志内容
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	logContent := string(content)

	// 验证 With 方法添加的固定字段
	if !strings.Contains(logContent, "service=test-service") ||
		!strings.Contains(logContent, "version=1.0.0") {
		t.Error("Log missing fields from With() method")
	}

	// 验证消息内容
	expectedMessages := []string{
		"service message",
		"context message",
		"chained message",
	}
	for _, msg := range expectedMessages {
		if !strings.Contains(logContent, msg) {
			t.Errorf("Expected message '%s' not found in log", msg)
		}
	}

	// 验证链式调用添加的字段
	if !strings.Contains(logContent, "component=handler") {
		t.Error("Log missing fields from chained With() call")
	}

	// 验证消息顺序和完整性
	lines := strings.Split(logContent, "\n")
	foundService := false
	foundContext := false
	foundChained := false

	for _, line := range lines {
		switch {
		case strings.Contains(line, "service message"):
			// 服务日志应该包含 service 和 version 字段
			if !strings.Contains(line, "service=test-service") || !strings.Contains(line, "version=1.0.0") {
				t.Error("Service log line missing required fields")
			}
			foundService = true

		case strings.Contains(line, "context message"):
			// 上下文日志应该保持基本格式
			if !strings.Contains(line, "level=INFO") {
				t.Error("Context log line missing level field")
			}
			foundContext = true

		case strings.Contains(line, "chained message"):
			// 链式调用日志应该同时包含 component 字段
			if !strings.Contains(line, "component=handler") {
				t.Error("Chained log line missing component field")
			}
			foundChained = true
		}
	}

	if !foundService || !foundContext || !foundChained {
		t.Error("Not all expected log messages were found")
	}
}

// BenchmarkDailyRotateWriter benchmarks the DailyRotateWriter performance.
func BenchmarkDailyRotateWriter(b *testing.B) {
	tempDir := b.TempDir()
	basePath := filepath.Join(tempDir, "bench.log")

	drw, err := NewDailyRotateWriter(basePath)
	if err != nil {
		b.Fatalf("Failed to create DailyRotateWriter: %v", err)
	}
	defer drw.Close()

	testData := []byte("benchmark test message\n")

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

// BenchmarkSlogLogger benchmarks the SlogLogger performance.
func BenchmarkSlogLogger(b *testing.B) {
	tempDir := b.TempDir()
	logPath := filepath.Join(tempDir, "bench.log")

	logger, err := NewSlogLogger(LevelInfo, "text", logPath, false)
	if err != nil {
		b.Fatalf("Failed to create SlogLogger: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(ctx, "benchmark message", "key", "value", "number", 42)
		}
	})
}
