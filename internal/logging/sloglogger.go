// Package logging provides a structured logging interface with daily rotation support.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DailyRotateWriter implements daily log rotation by appending date to filename.
type DailyRotateWriter struct {
	basePath string
	file     *os.File
	lastDate string
	mu       sync.RWMutex
	stopCh   chan struct{} // 停止自动轮转的信号
	maxFiles int           // 最大保留的日志文件数量，0表示不限制
}

// SlogLogger wraps slog.Logger to satisfy the Logger interface.
type SlogLogger struct {
	base  *slog.Logger
	ctx   context.Context
	level Level
}

// NewDailyRotateWriter creates a writer that rotates logs daily.
// maxFiles specifies the maximum number of log files to keep (0 means unlimited).
func NewDailyRotateWriter(basePath string, maxFiles ...int) (*DailyRotateWriter, error) {
	maxFilesValue := 0 // 默认不限制
	if len(maxFiles) > 0 {
		maxFilesValue = maxFiles[0]
	}

	drw := &DailyRotateWriter{
		basePath: basePath,
		stopCh:   make(chan struct{}),
		maxFiles: maxFilesValue,
	}
	if err := drw.rotateIfNeeded(); err != nil {
		return nil, err
	}

	// 启动自动轮转检查
	go drw.autoRotate()

	return drw, nil
}

// NewSlogLogger constructs a SlogLogger with given output, format, level and daily rotation.
// format: "text" or "json"; output: "stdout", "stderr", or file path; daily: enable daily rotation.
// maxFiles: maximum number of log files to keep (0 means unlimited).
func NewSlogLogger(level Level, format, output string, daily bool, maxFiles ...int) (*SlogLogger, error) {
	maxFilesValue := 0
	if len(maxFiles) > 0 {
		maxFilesValue = maxFiles[0]
	}
	var w io.Writer
	switch strings.ToLower(output) {
	case "", "stdout":
		w = os.Stdout
	case "stderr":
		w = os.Stderr
	default:
		if daily {
			drw, err := NewDailyRotateWriter(output, maxFilesValue)
			if err != nil {
				return nil, err
			}
			w = drw
		} else {
			f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return nil, err
			}
			w = f
		}
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level.toSlog()}
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	l := slog.New(handler)
	return &SlogLogger{base: l, level: level}, nil
}

// Write implements io.Writer interface for DailyRotateWriter.
func (drw *DailyRotateWriter) Write(p []byte) (n int, err error) {
	drw.mu.RLock()
	file := drw.file
	drw.mu.RUnlock()

	if file == nil {
		return 0, fmt.Errorf("log file not initialized")
	}

	return file.Write(p)
}

// Close closes the current log file and stops auto rotation.
func (drw *DailyRotateWriter) Close() error {
	// 停止自动轮转
	select {
	case <-drw.stopCh:
		// 已经关闭了
	default:
		close(drw.stopCh)
	}

	drw.mu.Lock()
	defer drw.mu.Unlock()

	if drw.file != nil {
		return drw.file.Close()
	}
	return nil
}

// With returns a child logger with preset fields.
func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{base: l.base.With(args...), ctx: l.ctx, level: l.level}
}

// WithContext binds default context to the logger.
func (l *SlogLogger) WithContext(ctx context.Context) Logger {
	return &SlogLogger{base: l.base, ctx: ctx, level: l.level}
}

// Enabled reports whether the specified level is enabled.
func (l *SlogLogger) Enabled(ctx context.Context, level Level) bool {
	if ctx == nil {
		ctx = l.ctx
	}
	return l.base.Enabled(ctx, level.toSlog())
}

// Debug logs a debug message.
func (l *SlogLogger) Debug(ctx context.Context, msg string, args ...any) {
	if ctx == nil {
		ctx = l.ctx
	}
	l.base.DebugContext(ctx, msg, args...)
}

// Info logs an info message.
func (l *SlogLogger) Info(ctx context.Context, msg string, args ...any) {
	if ctx == nil {
		ctx = l.ctx
	}
	l.base.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message.
func (l *SlogLogger) Warn(ctx context.Context, msg string, args ...any) {
	if ctx == nil {
		ctx = l.ctx
	}
	l.base.WarnContext(ctx, msg, args...)
}

// Error logs an error message.
func (l *SlogLogger) Error(ctx context.Context, msg string, args ...any) {
	if ctx == nil {
		ctx = l.ctx
	}
	l.base.ErrorContext(ctx, msg, args...)
}

// rotateIfNeeded checks if a new log file should be created for the current day.
func (drw *DailyRotateWriter) rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")
	drw.mu.Lock()
	defer drw.mu.Unlock()

	// 检查是否需要轮转
	if drw.file != nil && drw.lastDate == today {
		return nil
	}

	// 确定新文件名
	ext := filepath.Ext(drw.basePath)
	basename := drw.basePath
	if ext != "" {
		basename = drw.basePath[:len(drw.basePath)-len(ext)]
	}
	newFilename := fmt.Sprintf("%s-%s%s", basename, today, ext)
	if ext == "" {
		newFilename += ".log"
	}

	// Close existing file if open
	if drw.file != nil {
		if err := drw.file.Close(); err != nil {
			return fmt.Errorf("failed to close existing file: %w", err)
		}
		drw.file = nil
	}

	// 获取并检查目录权限
	dir := filepath.Dir(drw.basePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 检查目录是否可写（通过创建临时文件）
	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp_write_test_%d", time.Now().UnixNano()))
	if f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644); err != nil {
		return fmt.Errorf("directory is not writable: %w", err)
	} else {
		f.Close()
		os.Remove(tmpFile)
	}

	// 如果新文件已存在，检查是否可写
	if _, err := os.Stat(newFilename); err == nil {
		if f, err := os.OpenFile(newFilename, os.O_WRONLY|os.O_APPEND, 0); err != nil {
			return fmt.Errorf("existing log file is not writable: %w", err)
		} else {
			f.Close()
		}
	}

	// 创建新文件
	f, err := os.OpenFile(newFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	// 更新状态
	drw.file = f
	drw.lastDate = today

	// 清理旧日志文件（如果配置了最大文件数）
	if drw.maxFiles > 0 {
		if err := drw.cleanOldLogFiles(); err != nil {
			return fmt.Errorf("failed to clean old log files: %w", err)
		}
	}

	return nil
}

// cleanOldLogFiles 清理超过最大保留数量的旧日志文件
func (drw *DailyRotateWriter) cleanOldLogFiles() error {
	// 获取日志文件所在目录
	dir := filepath.Dir(drw.basePath)
	baseFileName := filepath.Base(drw.basePath)

	// 读取目录中的所有文件
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %v", err)
	}

	// 筛选出符合日志文件命名模式的文件
	ext := filepath.Ext(baseFileName)
	prefix := baseFileName
	if ext != "" {
		prefix = baseFileName[:len(baseFileName)-len(ext)]
	}
	var logFiles []string
	for _, file := range files {
		name := file.Name()
		if !file.IsDir() && strings.HasPrefix(name, prefix+"-") &&
			(strings.HasSuffix(name, ext) || (ext == "" && strings.HasSuffix(name, ".log"))) {
			logFiles = append(logFiles, filepath.Join(dir, name))
		}
	}

	// 如果日志文件数量未超过最大保留数量，则不需要清理
	if len(logFiles) <= drw.maxFiles {
		return nil
	}

	// 按文件修改时间排序
	sort.Slice(logFiles, func(i, j int) bool {
		infoI, _ := os.Stat(logFiles[i])
		infoJ, _ := os.Stat(logFiles[j])
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// 删除最旧的文件，直到文件数量等于最大保留数量
	for i := 0; i < len(logFiles)-drw.maxFiles; i++ {
		if err := os.Remove(logFiles[i]); err != nil {
			return fmt.Errorf("failed to remove old log file %s: %v", logFiles[i], err)
		}
	}
	return nil
}

// autoRotate runs in a goroutine to check for daily rotation.
func (drw *DailyRotateWriter) autoRotate() {
	ticker := time.NewTicker(time.Minute) // 每分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查是否需要轮转
			_ = drw.rotateIfNeeded() // 忽略轮转错误，不中断服务
		case <-drw.stopCh:
			// 收到停止信号，退出 goroutine
			return
		}
	}
}

// toSlog converts Level to slog.Level.
func (lvl Level) toSlog() slog.Level {
	switch lvl {
	case LevelDebug:
		return slog.LevelDebug
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
