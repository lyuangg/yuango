package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/lyuangg/yuango/internal/config"
	"github.com/lyuangg/yuango/internal/logging"
)

// mockLogger 实现 logging.Logger 接口用于测试
type mockLogger struct {
	mu          sync.RWMutex
	infoCalled  bool
	errorCalled bool
	debugCalled bool
	warnCalled  bool
	lastMessage string
	lastArgs    []interface{}
}

func (m *mockLogger) With(args ...any) logging.Logger {
	return m
}

func (m *mockLogger) WithContext(ctx context.Context) logging.Logger {
	return m
}

func (m *mockLogger) Enabled(ctx context.Context, level logging.Level) bool {
	return true
}

func (m *mockLogger) Info(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoCalled = true
	m.lastMessage = msg
	m.lastArgs = args
}

func (m *mockLogger) Error(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalled = true
	m.lastMessage = msg
	m.lastArgs = args
}

func (m *mockLogger) Debug(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugCalled = true
	m.lastMessage = msg
	m.lastArgs = args
}

func (m *mockLogger) Warn(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warnCalled = true
	m.lastMessage = msg
	m.lastArgs = args
}

// 安全的读取方法
func (m *mockLogger) getInfoCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.infoCalled
}

func (m *mockLogger) getLastMessage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastMessage
}

func (m *mockLogger) getErrorCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorCalled
}

// ====================
// DefaultConfig 测试
// ====================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Port", cfg.Port, "8080"},
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, 5 * time.Second},
		{"ReadTimeout", cfg.ReadTimeout, 10 * time.Second},
		{"WriteTimeout", cfg.WriteTimeout, 15 * time.Second},
		{"IdleTimeout", cfg.IdleTimeout, 60 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("DefaultConfig().%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// ====================
// NewConfigFromAppConfig 测试
// ====================

func TestNewConfigFromAppConfig(t *testing.T) {
	appCfg := config.Config{
		Port: "9090",
		Server: config.ServerConfig{
			ReadHeaderTimeout: 3,
			ReadTimeout:       8,
			WriteTimeout:      12,
			IdleTimeout:       30,
			ShutdownTimeout:   5,
		},
	}

	cfg := NewConfigFromAppConfig(appCfg)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Port", cfg.Port, "9090"},
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, 3 * time.Second},
		{"ReadTimeout", cfg.ReadTimeout, 8 * time.Second},
		{"WriteTimeout", cfg.WriteTimeout, 12 * time.Second},
		{"IdleTimeout", cfg.IdleTimeout, 30 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("NewConfigFromAppConfig().%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// ====================
// New 测试
// ====================

func TestNew(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	logger := &mockLogger{}

	t.Run("with valid config", func(t *testing.T) {
		cfg := Config{
			Port:              "8080",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   10 * time.Second,
		}

		srv := New(handler, cfg, logger)

		if srv == nil {
			t.Fatal("New() returned nil")
		}

		if srv.httpServer == nil {
			t.Fatal("httpServer is nil")
		}

		if srv.logger == nil {
			t.Error("logger not set")
		}

		if srv.httpServer.Addr != ":8080" {
			t.Errorf("httpServer.Addr = %s, want :8080", srv.httpServer.Addr)
		}

		if srv.httpServer.Handler == nil {
			t.Error("handler not set")
		}

		if srv.httpServer.ReadHeaderTimeout != 5*time.Second {
			t.Errorf("ReadHeaderTimeout = %v, want 5s", srv.httpServer.ReadHeaderTimeout)
		}

		if srv.httpServer.ReadTimeout != 10*time.Second {
			t.Errorf("ReadTimeout = %v, want 10s", srv.httpServer.ReadTimeout)
		}

		if srv.httpServer.WriteTimeout != 15*time.Second {
			t.Errorf("WriteTimeout = %v, want 15s", srv.httpServer.WriteTimeout)
		}

		if srv.httpServer.IdleTimeout != 60*time.Second {
			t.Errorf("IdleTimeout = %v, want 60s", srv.httpServer.IdleTimeout)
		}
	})

	t.Run("with empty port", func(t *testing.T) {
		cfg := Config{
			Port: "",
		}

		srv := New(handler, cfg, logger)

		if srv.httpServer.Addr != ":8080" {
			t.Errorf("httpServer.Addr = %s, want :8080 (default)", srv.httpServer.Addr)
		}
	})

	t.Run("with custom port", func(t *testing.T) {
		cfg := Config{
			Port: "9090",
		}

		srv := New(handler, cfg, logger)

		if srv.httpServer.Addr != ":9090" {
			t.Errorf("httpServer.Addr = %s, want :9090", srv.httpServer.Addr)
		}
	})
}

// ====================
// Shutdown 测试
// ====================

func TestShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	logger := &mockLogger{}
	cfg := DefaultConfig()

	t.Run("shutdown idle server", func(t *testing.T) {
		srv := New(handler, cfg, logger)

		// 创建监听器以获取可用端口
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to create listener: %v", err)
		}
		defer listener.Close()

		// 启动服务器
		go func() {
			_ = srv.httpServer.Serve(listener)
		}()

		// 等待服务器启动
		time.Sleep(100 * time.Millisecond)

		// 关闭服务器
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err = srv.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown() error = %v, want nil", err)
		}
	})

	t.Run("shutdown with timeout", func(t *testing.T) {
		// 使用一个不会立即关闭的 handler
		slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
		})

		srv := New(slowHandler, cfg, logger)

		// 启动服务器
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to create listener: %v", err)
		}
		defer listener.Close()

		go func() {
			_ = srv.httpServer.Serve(listener)
		}()

		// 等待服务器启动
		time.Sleep(50 * time.Millisecond)

		// 发起一个慢请求
		requestStarted := make(chan struct{})
		go func() {
			close(requestStarted)
			_, _ = http.Get("http://" + listener.Addr().String())
		}()

		// 等待请求真正开始
		<-requestStarted
		time.Sleep(100 * time.Millisecond)

		// 使用很短的超时来强制超时
		shutdownStart := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = srv.Shutdown(ctx)
		shutdownDuration := time.Since(shutdownStart)

		// 严格验证：应该返回超时错误
		if err == nil {
			t.Error("Shutdown() expected timeout error, got nil")
		} else if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Shutdown() error = %v, expected context.DeadlineExceeded", err)
		}

		// 验证关闭时间接近超时时间（允许一定误差）
		if shutdownDuration < 50*time.Millisecond || shutdownDuration > 200*time.Millisecond {
			t.Logf("Shutdown duration = %v (expected ~100ms)", shutdownDuration)
		}
	})
}

// ====================
// ListenAndServe 测试
// ====================

func TestListenAndServe(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	t.Run("start and serve requests", func(t *testing.T) {
		// 创建 logger
		logger := &mockLogger{}

		// 先获取一个可用端口（最小化竞态窗口）
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to create listener: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		addr := listener.Addr().String()
		listener.Close()

		// 立即使用这个端口创建服务器
		cfg := Config{
			Port: fmt.Sprintf("%d", port),
		}
		srv := New(handler, cfg, logger)

		// 捕获启动错误
		errChan := make(chan error, 1)
		go func() {
			// 真正调用 srv.ListenAndServe() 方法
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- err
			}
		}()

		// 等待服务器启动或失败
		select {
		case err := <-errChan:
			t.Fatalf("server failed to start: %v", err)
		case <-time.After(100 * time.Millisecond):
			// 服务器应该已经启动
		}

		// 检查 logger 是否被调用（使用线程安全的方法）
		if !logger.getInfoCalled() {
			t.Error("logger.Info() was not called")
		}

		if msg := logger.getLastMessage(); msg != "server starting" {
			t.Errorf("logger message = %s, want 'server starting'", msg)
		}

		// 发送测试请求
		resp, err := http.Get("http://" + addr)
		if err != nil {
			t.Fatalf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		// 关闭服务器
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
}

// ====================
// Run 测试（集成测试）
// ====================

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Run test in short mode")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	logger := &mockLogger{}

	t.Run("run with signal shutdown", func(t *testing.T) {
		// 创建监听器以获取实际端口
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to create listener: %v", err)
		}
		actualPort := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		cfg := Config{
			Port:            fmt.Sprintf("%d", actualPort),
			ShutdownTimeout: 2 * time.Second,
		}

		srv := New(handler, cfg, logger)

		// 捕获 Run 的返回值
		runDone := make(chan error, 1)
		go func() {
			// 真正调用 srv.Run() 方法
			runDone <- srv.Run(2 * time.Second)
		}()

		// 等待服务器启动
		time.Sleep(200 * time.Millisecond)

		// 验证服务器运行正常
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d", actualPort))
		if err != nil {
			t.Fatalf("failed to connect to server: %v", err)
		}
		resp.Body.Close()

		// 发送中断信号触发优雅关闭
		proc, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatalf("failed to find process: %v", err)
		}

		if err := proc.Signal(syscall.SIGTERM); err != nil {
			t.Fatalf("failed to send signal: %v", err)
		}

		// 等待 Run 方法完成
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("Run() error = %v, want nil", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Run() did not complete in time")
		}

		// 验证服务器已关闭
		_, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d", actualPort))
		if err == nil {
			t.Error("expected error when connecting to shut down server")
		}

		// 验证日志调用（使用线程安全的方法）
		if !logger.getInfoCalled() {
			t.Error("logger.Info() was not called")
		}
	})
}

// ====================
// 集成测试：完整的服务器生命周期
// ====================

func TestServerLifecycle(t *testing.T) {
	requestCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	logger := &mockLogger{}
	cfg := DefaultConfig()

	// 创建监听器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	srv := New(handler, cfg, logger)
	srv.httpServer.Addr = addr

	// 启动服务器
	go func() {
		_ = srv.ListenAndServe()
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 发送多个请求
	for i := 0; i < 5; i++ {
		resp, err := http.Get("http://" + addr)
		if err != nil {
			t.Logf("request %d failed: %v (server may be shutting down)", i, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("request %d: status = %d, want 200", i, resp.StatusCode)
		}
	}

	// 验证请求被处理
	if requestCount == 0 {
		t.Error("no requests were handled")
	}

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	// 验证服务器已关闭
	_, err = http.Get("http://" + addr)
	if err == nil {
		t.Error("expected error when connecting to shut down server")
	}
}

// ====================
// 基准测试
// ====================

func BenchmarkNew(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	logger := &mockLogger{}
	cfg := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New(handler, cfg, logger)
	}
}

func BenchmarkDefaultConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

func BenchmarkNewConfigFromAppConfig(b *testing.B) {
	appCfg := config.Config{
		Port: "8080",
		Server: config.ServerConfig{
			ReadHeaderTimeout: 5,
			ReadTimeout:       10,
			WriteTimeout:      15,
			IdleTimeout:       60,
			ShutdownTimeout:   10,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewConfigFromAppConfig(appCfg)
	}
}

// ====================
// 示例测试
// ====================

func ExampleNew() {
	// 创建一个简单的 handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 创建 logger（这里使用 mock）
	logger := &mockLogger{}

	// 创建配置
	cfg := DefaultConfig()
	cfg.Port = "8080"

	// 创建服务器
	srv := New(handler, cfg, logger)

	// 启动服务器（实际使用中）
	_ = srv
	// go srv.Run(cfg.ShutdownTimeout)

	// Output:
}

func ExampleDefaultConfig() {
	cfg := DefaultConfig()

	// 修改配置
	cfg.Port = "9090"
	cfg.ReadTimeout = 20 * time.Second

	_ = cfg
	// Output:
}
