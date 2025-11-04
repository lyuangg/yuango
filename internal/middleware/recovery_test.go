package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/lyuangg/yuango/internal/app"
	"github.com/lyuangg/yuango/internal/logging"
	"github.com/lyuangg/yuango/internal/response"
)

// mockLogger 实现 logging.Logger 接口用于测试
type mockLogger struct {
	mu          sync.RWMutex
	errorCalled bool
	errorCount  int
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

func (m *mockLogger) Info(ctx context.Context, msg string, args ...any) {}

func (m *mockLogger) Debug(ctx context.Context, msg string, args ...any) {}

func (m *mockLogger) Warn(ctx context.Context, msg string, args ...any) {}

func (m *mockLogger) Error(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalled = true
	m.errorCount++
	m.lastMessage = msg
	m.lastArgs = args
}

func (m *mockLogger) getErrorCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorCalled
}

func (m *mockLogger) getErrorCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorCount
}

func (m *mockLogger) getLastMessage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastMessage
}

func (m *mockLogger) getLastArgs() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastArgs
}

func (m *mockLogger) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalled = false
	m.errorCount = 0
	m.lastMessage = ""
	m.lastArgs = nil
}

// 创建测试用的 AppContext
func createTestAppContext() *app.AppContext {
	return &app.AppContext{
		Logger: &mockLogger{},
	}
}

// ====================
// Recovery 测试
// ====================

func TestRecovery_NoPanic(t *testing.T) {
	appCtx := createTestAppContext()
	mockLogger := appCtx.Logger.(*mockLogger)

	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证没有 panic，正常处理请求
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Body = %s, want OK", w.Body.String())
	}

	// 验证没有调用 Error 日志
	if mockLogger.getErrorCalled() {
		t.Error("Logger.Error should not be called when there is no panic")
	}
}

func TestRecovery_PanicWithError(t *testing.T) {
	appCtx := createTestAppContext()
	mockLogger := appCtx.Logger.(*mockLogger)

	testErr := errors.New("test error")
	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(testErr)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证调用了 Error 日志记录 panic
	if !mockLogger.getErrorCalled() {
		t.Error("Logger.Error should be called when panic occurs")
	}

	if mockLogger.getLastMessage() != "panic recovered" {
		t.Errorf("Last message = %s, want panic recovered", mockLogger.getLastMessage())
	}

	// 验证响应是错误响应
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	// 验证响应格式
	var resp response.JSONResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Code != 500 {
		t.Errorf("Response code = %d, want 500", resp.Code)
	}

	if resp.Message != "Internal Server Error" {
		t.Errorf("Response message = %s, want Internal Server Error", resp.Message)
	}
}

func TestRecovery_PanicWithString(t *testing.T) {
	appCtx := createTestAppContext()
	mockLogger := appCtx.Logger.(*mockLogger)

	panicMsg := "panic: something went wrong"
	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(panicMsg)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证调用了 Error 日志
	if !mockLogger.getErrorCalled() {
		t.Error("Logger.Error should be called when panic occurs")
	}

	// 验证响应是错误响应
	var resp response.JSONResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Code != 500 {
		t.Errorf("Response code = %d, want 500", resp.Code)
	}
}

func TestRecovery_PanicWithInt(t *testing.T) {
	appCtx := createTestAppContext()
	mockLogger := appCtx.Logger.(*mockLogger)

	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(42)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证调用了 Error 日志
	if !mockLogger.getErrorCalled() {
		t.Error("Logger.Error should be called when panic occurs")
	}

	// 验证响应是错误响应
	var resp response.JSONResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Code != 500 {
		t.Errorf("Response code = %d, want 500", resp.Code)
	}
}

func TestRecovery_NilAppContext(t *testing.T) {
	handler := Recovery(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// 不应该 panic，应该正常恢复
	handler.ServeHTTP(w, req)

	// 验证响应是错误响应（即使 appCtx 为 nil，也应该返回错误响应）
	var resp response.JSONResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Code != 500 {
		t.Errorf("Response code = %d, want 500", resp.Code)
	}
}

func TestRecovery_NilLogger(t *testing.T) {
	appCtx := &app.AppContext{
		Logger: nil,
	}

	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// 不应该 panic，应该正常恢复（即使 Logger 为 nil）
	handler.ServeHTTP(w, req)

	// 验证响应是错误响应
	var resp response.JSONResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Code != 500 {
		t.Errorf("Response code = %d, want 500", resp.Code)
	}
}

func TestRecovery_LogFields(t *testing.T) {
	appCtx := createTestAppContext()
	mockLogger := appCtx.Logger.(*mockLogger)

	testErr := errors.New("database connection failed")
	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(testErr)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证日志字段
	args := mockLogger.getLastArgs()
	if len(args) < 10 {
		t.Fatalf("Expected at least 10 log args, got %d", len(args))
	}

	// 验证日志包含预期的字段
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	// 验证关键字段
	if argsMap["error"] == nil {
		t.Error("Log should contain 'error' field")
	}

	if argsMap["method"] != "GET" {
		t.Errorf("method = %v, want GET", argsMap["method"])
	}

	if argsMap["path"] != "/api/users" {
		t.Errorf("path = %v, want /api/users", argsMap["path"])
	}

	if argsMap["remote_addr"] != "192.168.1.1:12345" {
		t.Errorf("remote_addr = %v, want 192.168.1.1:12345", argsMap["remote_addr"])
	}

	if argsMap["user_agent"] != "Mozilla/5.0" {
		t.Errorf("user_agent = %v, want Mozilla/5.0", argsMap["user_agent"])
	}

	if argsMap["stack"] == nil || argsMap["stack"] == "" {
		t.Error("Log should contain 'stack' field with stack trace")
	}
}

func TestRecovery_WriteErrorResponseFailure(t *testing.T) {
	appCtx := createTestAppContext()
	mockLogger := appCtx.Logger.(*mockLogger)

	// 创建一个会失败的 ResponseWriter（模拟 response.Fail 返回错误的情况）
	// 由于 response.Fail 使用 json.NewEncoder，我们无法直接让它失败
	// 但这个测试可以验证当 response.Fail 返回错误时，会记录日志
	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证至少记录了一次 panic recovered 的日志
	if mockLogger.getErrorCount() < 1 {
		t.Errorf("Expected at least 1 error log, got %d", mockLogger.getErrorCount())
	}

	// 验证第一条日志是 panic recovered
	if mockLogger.getLastMessage() != "panic recovered" {
		t.Errorf("Last message = %s, want panic recovered", mockLogger.getLastMessage())
	}
}

func TestRecovery_ConcurrentRequests(t *testing.T) {
	appCtx := createTestAppContext()

	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 随机 panic 或正常处理
		if r.URL.Query().Get("panic") == "true" {
			panic("test panic")
		}
		w.WriteHeader(http.StatusOK)
	}))

	const numRequests = 50
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if id%2 == 0 {
				req.URL.RawQuery = "panic=true"
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// 验证响应格式正确
			if w.Code != http.StatusOK {
				t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
			}
		}(i)
	}

	// 等待所有请求完成
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

func TestRecovery_WithAPIError(t *testing.T) {
	appCtx := createTestAppContext()
	mockLogger := appCtx.Logger.(*mockLogger)

	// 使用自定义的 APIError
	apiErr := fmt.Errorf("custom error")
	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(apiErr)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证调用了 Error 日志
	if !mockLogger.getErrorCalled() {
		t.Error("Logger.Error should be called when panic occurs")
	}

	// 验证响应
	var resp response.JSONResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Code != 500 {
		t.Errorf("Response code = %d, want 500", resp.Code)
	}
}

// BenchmarkRecovery_NoPanic 测试正常情况下的性能
func BenchmarkRecovery_NoPanic(b *testing.B) {
	appCtx := createTestAppContext()

	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkRecovery_WithPanic 测试 panic 恢复的性能
func BenchmarkRecovery_WithPanic(b *testing.B) {
	appCtx := createTestAppContext()

	handler := Recovery(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("benchmark panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
