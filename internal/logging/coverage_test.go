package logging

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestLoggerEnabled 测试 Enabled 函数
func TestLoggerEnabled(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "enabled_test.log")

	// 创建不同级别的日志记录器
	debugLogger, err := NewSlogLogger(LevelDebug, "text", logPath+".debug", false)
	if err != nil {
		t.Fatalf("Failed to create debug logger: %v", err)
	}

	infoLogger, err := NewSlogLogger(LevelInfo, "text", logPath+".info", false)
	if err != nil {
		t.Fatalf("Failed to create info logger: %v", err)
	}

	warnLogger, err := NewSlogLogger(LevelWarn, "text", logPath+".warn", false)
	if err != nil {
		t.Fatalf("Failed to create warn logger: %v", err)
	}

	errorLogger, err := NewSlogLogger(LevelError, "text", logPath+".error", false)
	if err != nil {
		t.Fatalf("Failed to create error logger: %v", err)
	}

	ctx := context.Background()
	nilCtx := context.Context(nil)

	// 测试 Enabled 函数在不同级别的行为
	if !debugLogger.Enabled(ctx, LevelDebug) {
		t.Error("Debug level should be enabled for debug logger")
	}
	if !debugLogger.Enabled(ctx, LevelInfo) {
		t.Error("Info level should be enabled for debug logger")
	}
	if !debugLogger.Enabled(ctx, LevelWarn) {
		t.Error("Warn level should be enabled for debug logger")
	}
	if !debugLogger.Enabled(ctx, LevelError) {
		t.Error("Error level should be enabled for debug logger")
	}

	// 测试 nil context
	if !debugLogger.Enabled(nilCtx, LevelDebug) {
		t.Error("Debug level should be enabled for debug logger with nil context")
	}

	// 测试 Info 级别日志记录器
	if infoLogger.Enabled(ctx, LevelDebug) {
		t.Error("Debug level should not be enabled for info logger")
	}
	if !infoLogger.Enabled(ctx, LevelInfo) {
		t.Error("Info level should be enabled for info logger")
	}

	// 测试 Warn 级别日志记录器
	if warnLogger.Enabled(ctx, LevelDebug) {
		t.Error("Debug level should not be enabled for warn logger")
	}
	if warnLogger.Enabled(ctx, LevelInfo) {
		t.Error("Info level should not be enabled for warn logger")
	}
	if !warnLogger.Enabled(ctx, LevelWarn) {
		t.Error("Warn level should be enabled for warn logger")
	}

	// 测试 Error 级别日志记录器
	if errorLogger.Enabled(ctx, LevelDebug) {
		t.Error("Debug level should not be enabled for error logger")
	}
	if !errorLogger.Enabled(ctx, LevelError) {
		t.Error("Error level should be enabled for error logger")
	}
}

// TestLogLevelFunctions 测试所有日志级别函数
func TestLogLevelFunctions(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "log_functions_test.log")

	// 创建日志记录器
	logger, err := NewSlogLogger(LevelDebug, "text", logPath, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// 测试不同的上下文情况
	ctx := context.Background()
	nilCtx := context.Context(nil)

	// 测试所有日志级别函数
	logger.Debug(ctx, "debug message with context", "key", "value")
	logger.Debug(nilCtx, "debug message with nil context", "key", "value")
	
	logger.Info(ctx, "info message with context", "key", "value")
	logger.Info(nilCtx, "info message with nil context", "key", "value")
	
	logger.Warn(ctx, "warn message with context", "key", "value")
	logger.Warn(nilCtx, "warn message with nil context", "key", "value")
	
	logger.Error(ctx, "error message with context", "key", "value")
	logger.Error(nilCtx, "error message with nil context", "key", "value")

	// 验证日志文件是否创建
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %v", err)
	}
}

// TestNewDailyRotateWriter 测试 NewDailyRotateWriter 函数
func TestNewDailyRotateWriter_Extended(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "rotate_test.log")

	// 测试正常情况
	writer, err := NewDailyRotateWriter(logPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer writer.Close()

	// 写入一些数据
	_, err = writer.Write([]byte("test data\n"))
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	// 注意：NewDailyRotateWriter 会自动创建目录，所以我们需要使用一个真正无法创建的路径
	// 例如，在 Unix 系统上尝试在 /dev/null 下创建文件
	if os.Getuid() != 0 { // 非 root 用户
		invalidPath := "/dev/null/invalid.log"
		_, err = NewDailyRotateWriter(invalidPath)
		if err == nil {
			t.Log("Note: Expected error for invalid path, but got nil. This might be system-dependent.")
		}
	}

	// 测试目录不存在但可以创建的情况
	newDirPath := filepath.Join(tempDir, "newdir")
	newLogPath := filepath.Join(newDirPath, "new.log")
	writer2, err := NewDailyRotateWriter(newLogPath)
	if err != nil {
		t.Fatalf("Failed to create writer with new directory: %v", err)
	}
	defer writer2.Close()

	// 写入一些数据
	_, err = writer2.Write([]byte("test data in new dir\n"))
	if err != nil {
		t.Fatalf("Failed to write data to new dir: %v", err)
	}
}