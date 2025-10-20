// Package server provides HTTP server configuration and lifecycle management.
package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lyuangg/yuango/internal/config"
	"github.com/lyuangg/yuango/internal/logging"
)

// Server wraps an http.Server with graceful shutdown support.
type Server struct {
	httpServer *http.Server
	logger     logging.Logger
}

// Config holds HTTP server configuration options.
type Config struct {
	Port              string        // Server port (e.g., "8080")
	ReadHeaderTimeout time.Duration // Maximum duration for reading request headers
	ReadTimeout       time.Duration // Maximum duration for reading entire request
	WriteTimeout      time.Duration // Maximum duration for writing response
	IdleTimeout       time.Duration // Maximum duration for keep-alive connections
	ShutdownTimeout   time.Duration // Maximum duration for graceful shutdown
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Port:              "8080",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   10 * time.Second,
	}
}

// NewConfigFromAppConfig creates a server Config from application config.
func NewConfigFromAppConfig(cfg config.Config) Config {
	return Config{
		Port:              cfg.Port,
		ReadHeaderTimeout: time.Duration(cfg.Server.ReadHeaderTimeout) * time.Second,
		ReadTimeout:       time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(cfg.Server.IdleTimeout) * time.Second,
		ShutdownTimeout:   time.Duration(cfg.Server.ShutdownTimeout) * time.Second,
	}
}

// New creates a new Server with the given handler, configuration, and logger.
func New(handler http.Handler, cfg Config, logger logging.Logger) *Server {
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return &Server{
		httpServer: srv,
		logger:     logger,
	}
}

// Run starts the HTTP server and blocks until an interrupt signal is received.
// It handles graceful shutdown automatically.
func (s *Server) Run(shutdownTimeout time.Duration) error {
	ctx := context.Background()

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		s.logger.Info(ctx, "server starting", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Wait for interrupt signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		s.logger.Error(ctx, "server error", "error", err)
		return err
	case sig := <-quit:
		s.logger.Info(ctx, "received shutdown signal", "signal", sig)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error(ctx, "shutdown error", "error", err)
		return err
	}

	s.logger.Info(ctx, "server stopped gracefully")
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// ListenAndServe starts the HTTP server (non-blocking, no graceful shutdown).
// For production use, prefer Run() method.
func (s *Server) ListenAndServe() error {
	s.logger.Info(context.Background(), "server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}
