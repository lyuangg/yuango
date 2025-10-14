package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// TestSlogLogger_WithHooks tests the hook functionality.
func TestSlogLogger_WithHooks(t *testing.T) {
	var buf bytes.Buffer

	// Define a hook to mask passwords.
	maskPasswordHook := func(r slog.Record) slog.Record {
		newAttrs := make([]slog.Attr, 0, r.NumAttrs())
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "password" {
				a.Value = slog.StringValue("******")
			}
			newAttrs = append(newAttrs, a)
			return true
		})
		newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		newRecord.AddAttrs(newAttrs...)
		return newRecord
	}

	// Define a hook to add a static field.
	addStaticFieldHook := func(r slog.Record) slog.Record {
		r.AddAttrs(slog.String("static_field", "static_value"))
		return r
	}

	hooks := []Hook{maskPasswordHook, addStaticFieldHook}

	logger, err := NewSlogLogger(LevelInfo, "text", &buf, "", 0, hooks...)
	if err != nil {
		t.Fatalf("Failed to create SlogLogger with hooks: %v", err)
	}

	ctx := context.Background()
	logger.Info(ctx, "login attempt", "user", "testuser", "password", "secret123")

	logOutput := buf.String()

	// Verify that the password is masked.
	if !strings.Contains(logOutput, "password=******") {
		t.Errorf("Expected password to be masked, but it was not. Log: %s", logOutput)
	}

	// Verify that the static field is added.
	if !strings.Contains(logOutput, "static_field=static_value") {
		t.Errorf("Expected static field to be added, but it was not. Log: %s", logOutput)
	}

	// Verify that the original password is not present.
	if strings.Contains(logOutput, "secret123") {
		t.Errorf("Original password should not be present in the log. Log: %s", logOutput)
	}
}
