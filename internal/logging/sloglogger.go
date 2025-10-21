// Package logging provides a structured logging interface with rotation support.
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

// RotateWriter implements log rotation by appending a timestamp to the filename.
type RotateWriter struct {
	basePath        string
	file            *os.File
	lastRotationTag string
	mu              sync.RWMutex
	stopCh          chan struct{} // Signal to stop auto-rotation.
	maxFiles        int           // Maximum number of log files to keep. 0 means unlimited.
	rotate          string        // Rotation schedule: "hourly", "daily".
}

// ContextFieldsFunc is a function that extracts key-value pairs from context.
// It should return a slice of alternating keys and values: [key1, value1, key2, value2, ...].
type ContextFieldsFunc func(context.Context) []any

// SlogLogger wraps slog.Logger to satisfy the Logger interface.
type SlogLogger struct {
	base              *slog.Logger
	ctx               context.Context
	level             Level
	contextFieldsFunc ContextFieldsFunc // Optional function to extract context fields
}

// NewRotateWriter creates a writer that rotates logs based on the specified schedule.
// rotate: "hourly", "daily".
// maxFiles specifies the maximum number of log files to keep (0 means unlimited).
func NewRotateWriter(basePath string, rotate string, maxFiles ...int) (*RotateWriter, error) {
	maxFilesValue := 0 // Default: no limit.
	if len(maxFiles) > 0 {
		maxFilesValue = maxFiles[0]
	}

	rw := &RotateWriter{
		basePath: basePath,
		stopCh:   make(chan struct{}),
		maxFiles: maxFilesValue,
		rotate:   rotate,
	}
	if err := rw.rotateIfNeeded(); err != nil {
		return nil, err
	}

	// Start the auto-rotation check goroutine.
	go rw.autoRotate()

	return rw, nil
}

// NewSlogLogger constructs a SlogLogger with given output, format, level, rotation, and hooks.
// format: "text" or "json".
// output: an io.Writer or a string ("stdout", "stderr", or a file path).
// rotate: "hourly", "daily", or empty to disable rotation.
// maxFiles: maximum number of log files to keep (0 means unlimited).
// hooks: a slice of functions to process log records.
func NewSlogLogger(level Level, format string, output interface{}, rotate string, maxFiles int, hooks ...Hook) (*SlogLogger, error) {
	var w io.Writer
	switch v := output.(type) {
	case io.Writer:
		w = v
	case string:
		switch strings.ToLower(v) {
		case "", "stdout":
			w = os.Stdout
		case "stderr":
			w = os.Stderr
		default:
			if rotate != "" {
				rw, err := NewRotateWriter(v, rotate, maxFiles)
				if err != nil {
					return nil, err
				}
				w = rw
			} else {
				f, err := os.OpenFile(v, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
				if err != nil {
					return nil, err
				}
				w = f
			}
		}
	default:
		return nil, fmt.Errorf("invalid output type: %T", output)
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level.toSlog()}
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	if len(hooks) > 0 {
		handler = &HookHandler{Handler: handler, hooks: hooks}
	}

	l := slog.New(handler)
	return &SlogLogger{base: l, level: level}, nil
}

// Write implements the io.Writer interface for RotateWriter.
func (rw *RotateWriter) Write(p []byte) (n int, err error) {
	rw.mu.RLock()
	file := rw.file
	rw.mu.RUnlock()

	if file == nil {
		return 0, fmt.Errorf("log file not initialized")
	}

	return file.Write(p)
}

// Close closes the current log file and stops auto-rotation.
func (rw *RotateWriter) Close() error {
	// Stop the auto-rotation goroutine.
	select {
	case <-rw.stopCh:
		// Already closed.
	default:
		close(rw.stopCh)
	}

	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// With returns a child logger with preset fields.
func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{
		base:              l.base.With(args...),
		ctx:               l.ctx,
		level:             l.level,
		contextFieldsFunc: l.contextFieldsFunc,
	}
}

// WithContext binds a default context to the logger.
func (l *SlogLogger) WithContext(ctx context.Context) Logger {
	return &SlogLogger{
		base:              l.base,
		ctx:               ctx,
		level:             l.level,
		contextFieldsFunc: l.contextFieldsFunc,
	}
}

// WithContextFields configures a function to extract fields from context.
// This function will be called for every log entry to automatically add context fields.
// Example:
//
//	logger.WithContextFields(func(ctx context.Context) []any {
//	    if traceID := trace.GetTraceID(ctx); traceID != "" {
//	        return []any{"trace_id", traceID}
//	    }
//	    return nil
//	})
func (l *SlogLogger) WithContextFields(fn ContextFieldsFunc) *SlogLogger {
	return &SlogLogger{
		base:              l.base,
		ctx:               l.ctx,
		level:             l.level,
		contextFieldsFunc: fn,
	}
}

// Enabled reports whether the specified level is enabled.
func (l *SlogLogger) Enabled(ctx context.Context, level Level) bool {
	if ctx == nil {
		ctx = l.ctx
	}
	return l.base.Enabled(ctx, level.toSlog())
}

// withContextFields extracts fields from context using the configured function.
func (l *SlogLogger) withContextFields(ctx context.Context, args []any) []any {
	if ctx == nil || l.contextFieldsFunc == nil {
		return args
	}

	// Call the configured function to extract context fields
	contextFields := l.contextFieldsFunc(ctx)
	if len(contextFields) == 0 {
		return args
	}

	// Merge context fields with provided args
	newArgs := make([]any, 0, len(contextFields)+len(args))
	newArgs = append(newArgs, contextFields...)
	newArgs = append(newArgs, args...)
	return newArgs
}

// Debug logs a debug message with automatic context field extraction.
func (l *SlogLogger) Debug(ctx context.Context, msg string, args ...any) {
	if ctx == nil {
		ctx = l.ctx
	}
	args = l.withContextFields(ctx, args)
	l.base.DebugContext(ctx, msg, args...)
}

// Info logs an info message with automatic context field extraction.
func (l *SlogLogger) Info(ctx context.Context, msg string, args ...any) {
	if ctx == nil {
		ctx = l.ctx
	}
	args = l.withContextFields(ctx, args)
	l.base.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message with automatic context field extraction.
func (l *SlogLogger) Warn(ctx context.Context, msg string, args ...any) {
	if ctx == nil {
		ctx = l.ctx
	}
	args = l.withContextFields(ctx, args)
	l.base.WarnContext(ctx, msg, args...)
}

// Error logs an error message with automatic context field extraction.
func (l *SlogLogger) Error(ctx context.Context, msg string, args ...any) {
	if ctx == nil {
		ctx = l.ctx
	}
	args = l.withContextFields(ctx, args)
	l.base.ErrorContext(ctx, msg, args...)
}

// rotateIfNeeded checks if a new log file should be created based on the rotation schedule.
func (rw *RotateWriter) rotateIfNeeded() error {
	format := ""
	switch rw.rotate {
	case "hourly":
		format = "2006-01-02-15"
	case "daily":
		format = "2006-01-02"
	default:
		return nil // No rotation needed if schedule is not set or unknown.
	}

	currentTag := time.Now().Format(format)
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Check if rotation is needed.
	if rw.file != nil && rw.lastRotationTag == currentTag {
		return nil
	}

	// Determine the new filename.
	ext := filepath.Ext(rw.basePath)
	basename := rw.basePath
	if ext != "" {
		basename = rw.basePath[:len(rw.basePath)-len(ext)]
	}
	newFilename := fmt.Sprintf("%s-%s%s", basename, currentTag, ext)
	if ext == "" {
		newFilename += ".log"
	}

	// Close existing file if open.
	if rw.file != nil {
		if err := rw.file.Close(); err != nil {
			return fmt.Errorf("failed to close existing file: %w", err)
		}
		rw.file = nil
	}

	// Get and check directory permissions.
	dir := filepath.Dir(rw.basePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if the directory is writable by creating a temporary file.
	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp_write_test_%d", time.Now().UnixNano()))
	if f, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644); err != nil {
		return fmt.Errorf("directory is not writable: %w", err)
	} else {
		f.Close()
		os.Remove(tmpFile)
	}

	// If the new file already exists, check if it's writable.
	if _, err := os.Stat(newFilename); err == nil {
		if f, err := os.OpenFile(newFilename, os.O_WRONLY|os.O_APPEND, 0); err != nil {
			return fmt.Errorf("existing log file is not writable: %w", err)
		} else {
			f.Close()
		}
	}

	// Create the new log file.
	f, err := os.OpenFile(newFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	// Update state.
	rw.file = f
	rw.lastRotationTag = currentTag

	// Clean up old log files if maxFiles is set.
	if rw.maxFiles > 0 {
		if err := rw.cleanOldLogFiles(); err != nil {
			// Log this error but don't fail the rotation.
			// A separate monitoring mechanism should handle cleanup failures.
			fmt.Fprintf(os.Stderr, "failed to clean old log files: %v\n", err)
		}
	}

	return nil
}

// cleanOldLogFiles removes old log files exceeding the maxFiles limit.
func (rw *RotateWriter) cleanOldLogFiles() error {
	dir := filepath.Dir(rw.basePath)
	baseFileName := filepath.Base(rw.basePath)

	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %v", err)
	}

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

	if len(logFiles) <= rw.maxFiles {
		return nil
	}

	sort.Slice(logFiles, func(i, j int) bool {
		infoI, _ := os.Stat(logFiles[i])
		infoJ, _ := os.Stat(logFiles[j])
		if infoI == nil || infoJ == nil {
			return false // Should not happen in normal operation
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	for i := 0; i < len(logFiles)-rw.maxFiles; i++ {
		if err := os.Remove(logFiles[i]); err != nil {
			// Log error but continue trying to remove other files.
			fmt.Fprintf(os.Stderr, "failed to remove old log file %s: %v\n", logFiles[i], err)
		}
	}
	return nil
}

// autoRotate runs in a goroutine to check for rotation.
func (rw *RotateWriter) autoRotate() {
	ticker := time.NewTicker(time.Minute) // Check every minute.
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if rotation is needed.
			_ = rw.rotateIfNeeded() // Ignore rotation errors to not interrupt service.
		case <-rw.stopCh:
			// Received stop signal, exit goroutine.
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
