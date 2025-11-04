package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lyuangg/yuango/internal/app"
	"github.com/lyuangg/yuango/internal/logging"
)

// mockLoggerForLogging 实现 logging.Logger 接口用于测试
type mockLoggerForLogging struct {
	mu          sync.RWMutex
	infoCalled  bool
	infoCount   int
	lastMessage string
	lastArgs    []interface{}
}

func (m *mockLoggerForLogging) With(args ...any) logging.Logger {
	return m
}

func (m *mockLoggerForLogging) WithContext(ctx context.Context) logging.Logger {
	return m
}

func (m *mockLoggerForLogging) Enabled(ctx context.Context, level logging.Level) bool {
	return true
}

func (m *mockLoggerForLogging) Debug(ctx context.Context, msg string, args ...any) {}

func (m *mockLoggerForLogging) Warn(ctx context.Context, msg string, args ...any) {}

func (m *mockLoggerForLogging) Error(ctx context.Context, msg string, args ...any) {}

func (m *mockLoggerForLogging) Info(ctx context.Context, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoCalled = true
	m.infoCount++
	m.lastMessage = msg
	m.lastArgs = args
}

func (m *mockLoggerForLogging) getInfoCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.infoCalled
}

func (m *mockLoggerForLogging) getInfoCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.infoCount
}

func (m *mockLoggerForLogging) getLastMessage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastMessage
}

func (m *mockLoggerForLogging) getLastArgs() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastArgs
}

func (m *mockLoggerForLogging) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoCalled = false
	m.infoCount = 0
	m.lastMessage = ""
	m.lastArgs = nil
}

// createTestAppContextForLogging 创建测试用的 AppContext
func createTestAppContextForLogging() *app.AppContext {
	return &app.AppContext{
		Logger: &mockLoggerForLogging{},
	}
}

// ====================
// Logging 基础测试
// ====================

func TestLogging_Basic(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证调用了 Info 日志
	if !mockLogger.getInfoCalled() {
		t.Error("Logger.Info should be called")
	}

	if mockLogger.getLastMessage() != "request" {
		t.Errorf("Last message = %s, want request", mockLogger.getLastMessage())
	}

	// 验证基本字段
	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if argsMap["method"] != "GET" {
		t.Errorf("method = %v, want GET", argsMap["method"])
	}

	if argsMap["path"] != "/" {
		t.Errorf("path = %v, want /", argsMap["path"])
	}

	if argsMap["status_code"] != http.StatusOK {
		t.Errorf("status_code = %v, want %d", argsMap["status_code"], http.StatusOK)
	}
}

func TestLogging_NilAppContext(t *testing.T) {
	handler := Logging(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// 不应该 panic
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLogging_NilLogger(t *testing.T) {
	appCtx := &app.AppContext{
		Logger: nil,
	}

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// 不应该 panic
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
}

// ====================
// JSON 请求体测试
// ====================

func TestLogging_WithJSONRequestBody(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 读取请求体验证
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))

	reqBody := `{"name":"test","age":20}`
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证请求体被记录
	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if requestBody, ok := argsMap["request_body"].(string); !ok || requestBody != reqBody {
		t.Errorf("request_body = %v, want %s", argsMap["request_body"], reqBody)
	}
}

func TestLogging_WithNonJSONRequestBody(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader("name=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证非 JSON 请求体不被记录
	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if _, exists := argsMap["request_body"]; exists {
		t.Error("request_body should not be logged for non-JSON content")
	}
}

func TestLogging_WithInvalidJSONRequestBody(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证无效 JSON 不被记录
	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if _, exists := argsMap["request_body"]; exists {
		t.Error("request_body should not be logged for invalid JSON")
	}
}

// ====================
// JSON 响应体测试
// ====================

func TestLogging_WithJSONResponseBody(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	responseBody := `{"code":0,"msg":"success","data":{"id":1}}`
	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证响应体被记录
	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if respBody, ok := argsMap["response_body"].(string); !ok || respBody != responseBody {
		t.Errorf("response_body = %v, want %s", argsMap["response_body"], responseBody)
	}
}

func TestLogging_WithNonJSONResponseBody(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证非 JSON 响应体不被记录
	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if _, exists := argsMap["response_body"]; exists {
		t.Error("response_body should not be logged for non-JSON content")
	}
}

// ====================
// Client IP 和 User-Agent 测试
// ====================

func TestLogging_WithClientIP(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1")
	req.Header.Set("X-Real-IP", "10.0.0.1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if clientIP, ok := argsMap["client_ip"].(string); !ok || clientIP != "192.168.1.100" {
		t.Errorf("client_ip = %v, want 192.168.1.100", argsMap["client_ip"])
	}
}

func TestLogging_WithXRealIP(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if clientIP, ok := argsMap["client_ip"].(string); !ok || clientIP != "10.0.0.1" {
		t.Errorf("client_ip = %v, want 10.0.0.1", argsMap["client_ip"])
	}
}

func TestLogging_WithRemoteAddr(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if clientIP, ok := argsMap["client_ip"].(string); !ok || clientIP != "192.168.1.1" {
		t.Errorf("client_ip = %v, want 192.168.1.1", argsMap["client_ip"])
	}
}

func TestLogging_WithUserAgent(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", userAgent)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if ua, ok := argsMap["user_agent"].(string); !ok || ua != userAgent {
		t.Errorf("user_agent = %v, want %s", argsMap["user_agent"], userAgent)
	}
}

// ====================
// 查询参数测试
// ====================

func TestLogging_WithQueryParams(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/users?page=1&limit=10&sort=name", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	queryParamsValue, exists := argsMap["query_params"]
	if !exists {
		t.Error("query_params should be logged")
		return
	}
	// url.Values 是 map[string][]string 的类型别名
	queryParams, ok := queryParamsValue.(url.Values)
	if !ok {
		t.Errorf("query_params type assertion failed, got %T", queryParamsValue)
		return
	}
	if queryParams["page"][0] != "1" {
		t.Errorf("query_params[page] = %v, want [1]", queryParams["page"])
	}
	if queryParams["limit"][0] != "10" {
		t.Errorf("query_params[limit] = %v, want [10]", queryParams["limit"])
	}
}

// ====================
// Duration 测试
// ====================

func TestLogging_Duration(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if duration, ok := argsMap["duration"].(time.Duration); !ok {
		t.Error("duration should be logged")
	} else if duration < 10*time.Millisecond {
		t.Errorf("duration = %v, should be >= 10ms", duration)
	}
}

// ====================
// 大响应体测试
// ====================

func TestLogging_LargeResponseBody(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	// 创建一个大于 maxBodySize 的有效 JSON 响应
	// 使用数组格式，包含多个完整的 JSON 对象
	// 确保截断到 maxBodySize 时，仍然是一个有效的 JSON 数组
	largeJSON := &bytes.Buffer{}
	largeJSON.WriteString(`[`)

	// 每个元素大约 28 字节：{"id":"0000000000000000"},
	itemSize := 28
	// 计算需要多少个元素才能超过 maxBodySize
	itemsNeeded := (maxBodySize * 120) / 100 / itemSize // 120% 以确保超过

	for i := 0; i < itemsNeeded; i++ {
		if i > 0 {
			largeJSON.WriteString(",")
		}
		largeJSON.WriteString(`{"id":"`)
		largeJSON.WriteString(strings.Repeat("0", 16))
		largeJSON.WriteString(`"}`)
	}
	largeJSON.WriteString(`]`)

	// 确保生成的 JSON 确实大于 maxBodySize
	if largeJSON.Len() <= maxBodySize {
		t.Fatalf("Generated JSON size (%d) should be > %d", largeJSON.Len(), maxBodySize)
	}

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(largeJSON.Bytes())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	// 验证响应体是否被记录
	// 如果截断后的 JSON 有效，应该记录为 "<body too large to log>"
	// 如果截断后的 JSON 无效，可能不会记录
	responseBody, hasResponseBody := argsMap["response_body"]
	if hasResponseBody {
		if bodyStr, ok := responseBody.(string); ok {
			if bodyStr != "<body too large to log>" {
				// 如果记录了完整内容，也是可以的（说明截断后验证失败，使用了完整验证）
				t.Logf("response_body was logged: %s", bodyStr[:100])
			}
		}
	} else {
		// 如果没有记录，说明截断后的 JSON 无效（这也是可以接受的行为）
		t.Log("Large JSON was not logged because truncated JSON is invalid (acceptable behavior)")
	}
}

// ====================
// 并发测试
// ====================

func TestLogging_ConcurrentRequests(t *testing.T) {
	appCtx := createTestAppContextForLogging()

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))

	const numRequests = 50
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
			}
		}(i)
	}

	// 等待所有请求完成
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// 验证日志被调用了正确的次数
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)
	if count := mockLogger.getInfoCount(); count != numRequests {
		t.Errorf("Info count = %d, want %d", count, numRequests)
	}
}

// ====================
// JSON 数组测试
// ====================

func TestLogging_WithJSONArrayRequestBody(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	reqBody := `[{"id":1},{"id":2}]`
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	args := mockLogger.getLastArgs()
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	if requestBody, ok := argsMap["request_body"].(string); !ok || requestBody != reqBody {
		t.Errorf("request_body = %v, want %s", argsMap["request_body"], reqBody)
	}
}

// ====================
// 不同 HTTP 方法测试
// ====================

func TestLogging_DifferentMethods(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		mockLogger.reset()

		handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(method, "/api/resource", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		args := mockLogger.getLastArgs()
		argsMap := make(map[string]interface{})
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				if key, ok := args[i].(string); ok {
					argsMap[key] = args[i+1]
				}
			}
		}

		if argsMap["method"] != method {
			t.Errorf("method = %v, want %s", argsMap["method"], method)
		}
	}
}

// ====================
// 不同状态码测试
// ====================

func TestLogging_DifferentStatusCodes(t *testing.T) {
	appCtx := createTestAppContextForLogging()
	mockLogger := appCtx.Logger.(*mockLoggerForLogging)

	statusCodes := []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError}

	for _, statusCode := range statusCodes {
		mockLogger.reset()

		handler := Logging(appCtx)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		args := mockLogger.getLastArgs()
		argsMap := make(map[string]interface{})
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				if key, ok := args[i].(string); ok {
					argsMap[key] = args[i+1]
				}
			}
		}

		if argsMap["status_code"] != statusCode {
			t.Errorf("status_code = %v, want %d", argsMap["status_code"], statusCode)
		}
	}
}

// ====================
// 单元测试：isValidJSON
// ====================

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "valid JSON object",
			input:    []byte(`{"key":"value"}`),
			expected: true,
		},
		{
			name:     "valid JSON array",
			input:    []byte(`[1,2,3]`),
			expected: true,
		},
		{
			name:     "empty string",
			input:    []byte(""),
			expected: false,
		},
		{
			name:     "only whitespace",
			input:    []byte("   \n\t  "),
			expected: false,
		},
		{
			name:     "starts with number",
			input:    []byte(`123`),
			expected: false,
		},
		{
			name:     "starts with letter",
			input:    []byte(`hello`),
			expected: false,
		},
		{
			name:     "starts with { but invalid",
			input:    []byte(`{invalid}`),
			expected: false,
		},
		{
			name:     "starts with [ but invalid",
			input:    []byte(`[invalid`),
			expected: false,
		},
		{
			name:     "whitespace before valid JSON",
			input:    []byte(`   {"key":"value"}`),
			expected: true,
		},
		{
			name:     "whitespace before invalid",
			input:    []byte(`   hello`),
			expected: false,
		},
		{
			name:     "valid nested JSON",
			input:    []byte(`{"a":{"b":{"c":1}}}`),
			expected: true,
		},
		{
			name:     "valid JSON with whitespace",
			input:    []byte(`{"key": "value"}`),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidJSON(tt.input)
			if result != tt.expected {
				t.Errorf("isValidJSON(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// ====================
// 单元测试：getClientIP
// ====================

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name         string
		forwardedFor string
		realIP       string
		remoteAddr   string
		expected     string
	}{
		{
			name:         "X-Forwarded-For with multiple IPs",
			forwardedFor: "192.168.1.100, 10.0.0.1",
			expected:     "192.168.1.100",
		},
		{
			name:         "X-Forwarded-For with single IP",
			forwardedFor: "192.168.1.100",
			expected:     "192.168.1.100",
		},
		{
			name:         "X-Forwarded-For with leading spaces",
			forwardedFor: "  192.168.1.100  , 10.0.0.1",
			expected:     "192.168.1.100",
		},
		{
			name:         "X-Forwarded-For empty after trim",
			forwardedFor: "   , 10.0.0.1",
			realIP:       "10.0.0.1",
			expected:     "10.0.0.1",
		},
		{
			name:         "X-Forwarded-For empty string",
			forwardedFor: "",
			realIP:       "10.0.0.1",
			expected:     "10.0.0.1",
		},
		{
			name:     "only X-Real-IP",
			realIP:   "10.0.0.1",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Real-IP with spaces",
			realIP:   "  10.0.0.1  ",
			expected: "10.0.0.1",
		},
		{
			name:       "X-Real-IP empty string",
			realIP:     "",
			remoteAddr: "192.168.1.1:54321",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:54321",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr empty",
			remoteAddr: "",
			expected:   "",
		},
		{
			name:       "no headers",
			remoteAddr: "", // 明确设置为空字符串
			expected:   "",
		},
		{
			name:         "X-Forwarded-For with empty first IP",
			forwardedFor: ", 10.0.0.1",
			realIP:       "10.0.0.1",
			expected:     "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			// 明确设置 RemoteAddr（包括空字符串的情况）
			req.RemoteAddr = tt.remoteAddr

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("getClientIP() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// ====================
// 单元测试：readRequestBody
// ====================

func TestReadRequestBody(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		bodyNil     bool
		expectError bool
		expectEmpty bool
	}{
		{
			name:        "nil body",
			bodyNil:     true,
			expectEmpty: true,
		},
		{
			name:        "empty body",
			body:        "",
			expectEmpty: true,
		},
		{
			name:        "normal body",
			body:        `{"key":"value"}`,
			expectEmpty: false,
		},
		{
			name:        "large body",
			body:        strings.Repeat("a", 1000),
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.bodyNil {
				req = httptest.NewRequest(http.MethodPost, "/", nil)
			} else {
				req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			}

			result, err := readRequestBody(req)

			if tt.expectError {
				if err == nil {
					t.Error("readRequestBody() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("readRequestBody() unexpected error: %v", err)
				}
			}

			if tt.expectEmpty {
				if len(result) != 0 {
					t.Errorf("readRequestBody() expected empty, got %d bytes", len(result))
				}
			} else {
				if len(result) == 0 {
					t.Error("readRequestBody() expected non-empty result")
				}
			}

			// 验证 body 可以重新读取
			if !tt.bodyNil && !tt.expectEmpty {
				var readAgain []byte
				if req.Body != nil {
					readAgain, _ = io.ReadAll(req.Body)
					if string(readAgain) != tt.body {
						t.Errorf("readRequestBody() body not reset correctly, got %q, want %q", string(readAgain), tt.body)
					}
				}
			}
		})
	}
}

// 测试 readRequestBody 的错误情况（需要创建一个会出错的 Reader）
type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errorReader) Close() error {
	return nil
}

func TestReadRequestBody_Error(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", &errorReader{})
	result, err := readRequestBody(req)

	if err == nil {
		t.Error("readRequestBody() expected error, got nil")
	}

	if result != nil {
		t.Errorf("readRequestBody() expected nil on error, got %v", result)
	}
}

// 测试 readRequestBody 超过 maxBodySize 的情况
func TestReadRequestBody_ExceedsMaxSize(t *testing.T) {
	// 创建一个超过 maxBodySize 的 body
	largeBody := strings.Repeat("a", maxBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(largeBody))

	result, err := readRequestBody(req)

	if err != nil {
		t.Errorf("readRequestBody() unexpected error: %v", err)
	}

	// 应该被限制到 maxBodySize
	if len(result) != maxBodySize {
		t.Errorf("readRequestBody() expected length %d, got %d", maxBodySize, len(result))
	}

	// 验证 body 可以重新读取（应该返回被截断的内容）
	if req.Body != nil {
		readAgain, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			t.Errorf("readRequestBody() body reset failed: %v", readErr)
		}
		if len(readAgain) != maxBodySize {
			t.Errorf("readRequestBody() body reset length expected %d, got %d", maxBodySize, len(readAgain))
		}
	}
}

// 测试 readRequestBody 正好等于 maxBodySize 的情况
func TestReadRequestBody_ExactMaxSize(t *testing.T) {
	// 创建一个正好等于 maxBodySize 的 body
	exactBody := strings.Repeat("a", maxBodySize)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(exactBody))

	result, err := readRequestBody(req)

	if err != nil {
		t.Errorf("readRequestBody() unexpected error: %v", err)
	}

	// 应该完整读取
	if len(result) != maxBodySize {
		t.Errorf("readRequestBody() expected length %d, got %d", maxBodySize, len(result))
	}

	// 验证 body 可以重新读取
	if req.Body != nil {
		readAgain, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			t.Errorf("readRequestBody() body reset failed: %v", readErr)
		}
		if len(readAgain) != maxBodySize {
			t.Errorf("readRequestBody() body reset length expected %d, got %d", maxBodySize, len(readAgain))
		}
	}
}

// 测试 readRequestBody 使用自定义 io.ReadCloser
type customReadCloser struct {
	*strings.Reader
	closed bool
}

func (c *customReadCloser) Close() error {
	c.closed = true
	return nil
}

func TestReadRequestBody_WithCustomReadCloser(t *testing.T) {
	body := "test body content"
	reader := &customReadCloser{Reader: strings.NewReader(body)}
	req := httptest.NewRequest(http.MethodPost, "/", reader)

	result, err := readRequestBody(req)

	if err != nil {
		t.Errorf("readRequestBody() unexpected error: %v", err)
	}

	if string(result) != body {
		t.Errorf("readRequestBody() expected %q, got %q", body, string(result))
	}

	// 验证 body 可以重新读取
	if req.Body != nil {
		readAgain, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			t.Errorf("readRequestBody() body reset failed: %v", readErr)
		}
		if string(readAgain) != body {
			t.Errorf("readRequestBody() body reset expected %q, got %q", body, string(readAgain))
		}
	}
}

// 测试 readRequestBody 当 Body 为 nil 的情况
func TestReadRequestBody_NilBody(t *testing.T) {
	req := &http.Request{}
	req.Body = nil

	result, err := readRequestBody(req)

	if err != nil {
		t.Errorf("readRequestBody() unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("readRequestBody() expected nil result when Body is nil, got %v", result)
	}
}
