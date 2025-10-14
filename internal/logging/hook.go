// Package logging provides a structured logging interface with rotation support.
package logging

import (
	"context"
	"log/slog"
)

// Hook is a function that can process or modify a log record before it is written.
// It receives a record and must return a record.
type Hook func(r slog.Record) slog.Record

// HookHandler is a slog.Handler that wraps another handler and applies a set of hooks.
type HookHandler struct {
	slog.Handler
	hooks []Hook
}

// Handle processes the log record by applying hooks and then passing it to the wrapped handler.
func (h *HookHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, hook := range h.hooks {
		r = hook(r)
	}
	return h.Handler.Handle(ctx, r)
}
