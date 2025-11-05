// Package database provides database connection management.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lyuangg/yuango/internal/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormLoggerAdapter adapts the application's Logger to GORM's logger.Interface.
type GormLoggerAdapter struct {
	logger        logging.Logger
	logLevel      logger.LogLevel
	slowThreshold time.Duration
}

// NewGormLoggerAdapter creates a new GORM logger adapter from the application logger.
func NewGormLoggerAdapter(appLogger logging.Logger, logLevel string) logger.Interface {
	level := parseLogLevel(logLevel)
	return &GormLoggerAdapter{
		logger:        appLogger,
		logLevel:      level,
		slowThreshold: time.Second, // Default slow query threshold
	}
}

// parseLogLevel parses the log level string to GORM's LogLevel.
func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	case "info", "":
		return logger.Info
	default:
		return logger.Info
	}
}

// LogMode sets the log level and returns a new logger instance.
func (l *GormLoggerAdapter) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info logs an info message.
func (l *GormLoggerAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		l.logger.Info(ctx, msg, data...)
	}
}

// Warn logs a warning message.
func (l *GormLoggerAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		l.logger.Warn(ctx, msg, data...)
	}
}

// Error logs an error message.
func (l *GormLoggerAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		l.logger.Error(ctx, msg, data...)
	}
}

// Trace logs SQL queries with timing information.
func (l *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	var logLevel logging.Level
	var logMsg string
	var logArgs []interface{}

	logArgs = append(logArgs, "duration", elapsed.String(), "rows", rows, "sql", sql)

	switch {
	case err != nil && l.logLevel >= logger.Error && (!errors.Is(err, gorm.ErrRecordNotFound)):
		logLevel = logging.LevelError
		logMsg = "SQL error"
		logArgs = append(logArgs, "error", err)
	case elapsed > l.slowThreshold && l.slowThreshold > 0 && l.logLevel >= logger.Warn:
		logLevel = logging.LevelWarn
		logMsg = fmt.Sprintf("slow SQL query (>%v)", l.slowThreshold)
	case l.logLevel >= logger.Info:
		logLevel = logging.LevelInfo
		logMsg = "SQL query"
	default:
		// Silent or lower level, don't log
		return
	}

	switch logLevel {
	case logging.LevelError:
		l.logger.Error(ctx, logMsg, logArgs...)
	case logging.LevelWarn:
		l.logger.Warn(ctx, logMsg, logArgs...)
	case logging.LevelInfo:
		l.logger.Info(ctx, logMsg, logArgs...)
	}
}
