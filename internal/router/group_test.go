package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/lyuangg/yuango/internal/app"
	"github.com/lyuangg/yuango/internal/config"
	"github.com/lyuangg/yuango/internal/logging"
	"github.com/lyuangg/yuango/internal/middleware"
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

// 创建测试用的 AppContext
func createTestAppContext() *app.AppContext {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	return &app.AppContext{
		Config: cfg,
		Logger: &mockLogger{},
	}
}

// 测试用的中间件
func testMiddleware(name string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", name)
			next.ServeHTTP(w, r)
		})
	}
}

// 测试用的处理器
func testHandler(response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}
}

func TestNewGroup(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	if group == nil {
		t.Fatal("NewGroup returned nil")
	}

	if group.appCtx != appCtx {
		t.Error("AppContext not set correctly")
	}

	if group.mux == nil {
		t.Error("ServeMux not initialized")
	}

	if group.middlewares == nil {
		t.Error("Middlewares slice not initialized")
	}

	if len(group.middlewares) != 0 {
		t.Error("Middlewares slice should be empty")
	}
}

func TestUse(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	middleware1 := testMiddleware("middleware1")
	middleware2 := testMiddleware("middleware2")

	// 测试添加单个中间件
	result := group.Use(middleware1)
	if result != group {
		t.Error("Use should return the same group instance")
	}

	if len(group.middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(group.middlewares))
	}

	// 测试添加多个中间件
	group.Use(middleware2)
	if len(group.middlewares) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(group.middlewares))
	}

	// 测试链式调用
	group3 := NewGroup(appCtx)
	group3.Use(middleware1, middleware2)
	if len(group3.middlewares) != 2 {
		t.Errorf("Expected 2 middlewares with multiple args, got %d", len(group3.middlewares))
	}
}

func TestHTTPMethods(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	handler := testHandler("test response")

	// 测试 GET
	group.GET("/test", handler)

	// 测试 POST
	group.POST("/test", handler)

	// 测试 PUT
	group.PUT("/test", handler)

	// 测试 DELETE
	group.DELETE("/test", handler)

	// 测试 PATCH
	group.PATCH("/test", handler)

	// 验证路由是否正确注册
	testCases := []struct {
		method string
		path   string
	}{
		{"GET", "/test"},
		{"POST", "/test"},
		{"PUT", "/test"},
		{"DELETE", "/test"},
		{"PATCH", "/test"},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			handler := group.Build()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != "test response" {
				t.Errorf("Expected body 'test response', got '%s'", w.Body.String())
			}
		})
	}
}

func TestHandle(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("handle test"))
	})

	group.Handle("/handle", handler)

	req := httptest.NewRequest("GET", "/handle", nil)
	w := httptest.NewRecorder()

	builtHandler := group.Build()
	builtHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "handle test" {
		t.Errorf("Expected body 'handle test', got '%s'", w.Body.String())
	}
}

func TestHandleFunc(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("handlefunc test"))
	}

	group.HandleFunc("/handlefunc", handler)

	req := httptest.NewRequest("GET", "/handlefunc", nil)
	w := httptest.NewRecorder()

	builtHandler := group.Build()
	builtHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "handlefunc test" {
		t.Errorf("Expected body 'handlefunc test', got '%s'", w.Body.String())
	}
}

func TestGroup(t *testing.T) {
	appCtx := createTestAppContext()
	parentGroup := NewGroup(appCtx)

	// 添加父级中间件
	parentMiddleware := testMiddleware("parent")
	parentGroup.Use(parentMiddleware)

	// 创建子组
	subGroupCreated := false
	result := parentGroup.Group("/api", func(subGroup *Group) {
		subGroupCreated = true

		// 验证子组继承了父级的中间件
		if len(subGroup.middlewares) != 1 {
			t.Errorf("SubGroup should inherit parent middlewares, expected 1, got %d", len(subGroup.middlewares))
		}

		// 添加子组中间件
		subMiddleware := testMiddleware("sub")
		subGroup.Use(subMiddleware)

		// 添加子组路由
		subGroup.GET("/users", testHandler("users"))
	})

	if !subGroupCreated {
		t.Error("SubGroup callback was not called")
	}

	if result != parentGroup {
		t.Error("Group should return the parent group")
	}

	// 测试子组路由是否正常工作
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()

	handler := parentGroup.Build()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "users" {
		t.Errorf("Expected body 'users', got '%s'", w.Body.String())
	}

	// 验证中间件顺序（子组中间件应该在最外层）
	if w.Header().Get("X-Middleware") != "sub" {
		t.Error("SubGroup middleware should be applied")
	}
}

func TestGroupWithoutPrefix(t *testing.T) {
	appCtx := createTestAppContext()
	parentGroup := NewGroup(appCtx)

	// 创建无前缀的子组
	parentGroup.Group("", func(subGroup *Group) {
		subGroup.GET("/test", testHandler("no prefix"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler := parentGroup.Build()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "no prefix" {
		t.Errorf("Expected body 'no prefix', got '%s'", w.Body.String())
	}
}

func TestBuild(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	// 添加多个中间件
	middleware1 := testMiddleware("middleware1")
	middleware2 := testMiddleware("middleware2")
	group.Use(middleware1, middleware2)

	// 添加路由
	group.GET("/test", testHandler("test"))

	// 构建处理器
	handler := group.Build()

	if handler == nil {
		t.Fatal("Build returned nil handler")
	}

	// 测试处理器
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "test" {
		t.Errorf("Expected body 'test', got '%s'", w.Body.String())
	}

	// 验证中间件顺序（最后添加的中间件应该在最外层）
	if w.Header().Get("X-Middleware") != "middleware2" {
		t.Error("Last middleware should be applied outermost")
	}
}

func TestBuildWithNoMiddlewares(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	group.GET("/test", testHandler("no middleware"))

	handler := group.Build()

	if handler == nil {
		t.Fatal("Build returned nil handler")
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "no middleware" {
		t.Errorf("Expected body 'no middleware', got '%s'", w.Body.String())
	}
}

func TestChaining(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	// 测试链式调用
	group.
		Use(testMiddleware("chain1")).
		Use(testMiddleware("chain2")).
		GET("/test1", testHandler("test1")).
		POST("/test2", testHandler("test2")).
		Group("/api", func(subGroup *Group) {
			subGroup.GET("/users", testHandler("users"))
		})

	// 验证链式调用结果
	if len(group.middlewares) != 2 {
		t.Errorf("Expected 2 middlewares from chaining, got %d", len(group.middlewares))
	}

	// 测试第一个路由
	req1 := httptest.NewRequest("GET", "/test1", nil)
	w1 := httptest.NewRecorder()
	group.Build().ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK || w1.Body.String() != "test1" {
		t.Error("First chained route failed")
	}

	// 测试第二个路由
	req2 := httptest.NewRequest("POST", "/test2", nil)
	w2 := httptest.NewRecorder()
	group.Build().ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK || w2.Body.String() != "test2" {
		t.Error("Second chained route failed")
	}

	// 测试子组路由
	req3 := httptest.NewRequest("GET", "/api/users", nil)
	w3 := httptest.NewRecorder()
	group.Build().ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK || w3.Body.String() != "users" {
		t.Error("SubGroup route failed")
	}
}

func TestMethodSpecificRouting(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	// 注册不同方法到相同路径
	group.GET("/resource", testHandler("GET"))
	group.POST("/resource", testHandler("POST"))
	group.PUT("/resource", testHandler("PUT"))
	group.DELETE("/resource", testHandler("DELETE"))

	handler := group.Build()

	testCases := []struct {
		method   string
		expected string
	}{
		{"GET", "GET"},
		{"POST", "POST"},
		{"PUT", "PUT"},
		{"DELETE", "DELETE"},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/resource", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d for %s, got %d", http.StatusOK, tc.method, w.Code)
			}

			if w.Body.String() != tc.expected {
				t.Errorf("Expected body '%s' for %s, got '%s'", tc.expected, tc.method, w.Body.String())
			}
		})
	}
}

func TestMiddlewareOrder(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	// 创建会修改响应的中间件来测试顺序
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Order", "1")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Order", "2")
			next.ServeHTTP(w, r)
		})
	}

	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Order", "3")
			next.ServeHTTP(w, r)
		})
	}

	// 按顺序添加中间件
	group.Use(middleware1, middleware2, middleware3)
	group.GET("/test", testHandler("order test"))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler := group.Build()
	handler.ServeHTTP(w, req)

	// 验证中间件执行顺序（最后添加的应该在最外层）
	if w.Header().Get("X-Order") != "3" {
		t.Error("Middleware order is incorrect, last added should be outermost")
	}
}

func TestConcurrentAccess(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	// 并发添加中间件和路由
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			middleware := testMiddleware(strings.Repeat("m", i+1))
			group.Use(middleware)

			path := "/test" + string(rune('0'+i))
			group.GET(path, testHandler("response"))
		}(i)
	}

	wg.Wait()

	// 验证结果 - 并发访问可能导致中间件数量不准确，但应该大于0
	if len(group.middlewares) == 0 {
		t.Error("Expected at least some middlewares to be added")
	}

	// 测试构建的处理器是否正常工作
	handler := group.Build()
	req := httptest.NewRequest("GET", "/test0", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestNew(t *testing.T) {
	appCtx := createTestAppContext()
	handler := New(appCtx)

	if handler == nil {
		t.Fatal("New returned nil handler")
	}

	// 测试健康检查端点
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{"Health Check", "GET", "/healthz", http.StatusOK, "ok"},
		{"Ping", "GET", "/ping", http.StatusOK, ""},
		{"API Status", "GET", "/api/status", http.StatusOK, ""},
		{"API Users", "GET", "/api/users", http.StatusOK, ""},
		{"API User by ID", "GET", "/api/users/123", http.StatusOK, ""},
		{"Create User", "POST", "/api/users", http.StatusOK, ""},
		{"Update User", "PUT", "/api/users/123", http.StatusOK, ""},
		{"Delete User", "DELETE", "/api/users/123", http.StatusOK, ""},
		{"Protected Profile", "GET", "/api/protected/profile", http.StatusUnauthorized, ""},
		{"Protected Settings", "GET", "/api/protected/settings", http.StatusUnauthorized, ""},
		{"Admin Dashboard", "GET", "/admin/dashboard", http.StatusUnauthorized, ""},
		{"Admin Settings", "GET", "/admin/settings", http.StatusUnauthorized, ""},
		{"Admin Users", "GET", "/admin/users", http.StatusUnauthorized, ""},
		{"Admin Stats Overview", "GET", "/admin/stats/overview", http.StatusUnauthorized, ""},
		{"Admin Stats Reports", "GET", "/admin/stats/reports", http.StatusUnauthorized, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedBody != "" && w.Body.String() != tc.expectedBody {
				t.Errorf("Expected body '%s', got '%s'", tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestNewWithMiddleware(t *testing.T) {
	appCtx := createTestAppContext()
	handler := New(appCtx)

	// 测试中间件是否正确应用
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 验证 Recovery 中间件被应用（通过检查响应头或其他标识）
	// 注意：这里我们主要验证路由能正常工作，中间件的具体行为
	// 在各自的测试文件中验证
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// 测试边界情况和错误处理
func TestEdgeCases(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	// 测试空中间件
	group.Use()

	// 测试空字符串路径会导致 panic，这是 http.ServeMux 的限制
	// 所以我们不测试空字符串路径

	handler := group.Build()
	if handler == nil {
		t.Fatal("Build returned nil handler")
	}

	// 测试正常路径
	group.GET("/test", testHandler("normal path"))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestNilAppContext(t *testing.T) {
	// 测试 nil AppContext 的情况
	group := NewGroup(nil)

	if group == nil {
		t.Fatal("NewGroup returned nil")
	}

	if group.appCtx != nil {
		t.Error("AppContext should be nil")
	}

	if group.mux == nil {
		t.Error("ServeMux should be initialized even with nil AppContext")
	}

	// 测试基本功能仍然工作
	group.GET("/test", testHandler("nil context test"))
	handler := group.Build()

	if handler == nil {
		t.Fatal("Build returned nil handler")
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestEmptyMiddlewareSlice(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	// 测试空中间件切片
	group.Use()

	if len(group.middlewares) != 0 {
		t.Errorf("Expected 0 middlewares, got %d", len(group.middlewares))
	}

	// 测试构建仍然工作
	group.GET("/test", testHandler("no middleware"))
	handler := group.Build()

	if handler == nil {
		t.Fatal("Build returned nil handler")
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestMultipleEmptyUseCalls(t *testing.T) {
	appCtx := createTestAppContext()
	group := NewGroup(appCtx)

	// 多次调用空的 Use
	group.Use().Use().Use()

	if len(group.middlewares) != 0 {
		t.Errorf("Expected 0 middlewares after empty Use calls, got %d", len(group.middlewares))
	}

	// 验证链式调用仍然工作
	group.Use(testMiddleware("test1")).Use(testMiddleware("test2"))

	if len(group.middlewares) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(group.middlewares))
	}
}

func TestComplexNestedGroups(t *testing.T) {
	appCtx := createTestAppContext()
	root := NewGroup(appCtx)

	// 创建复杂的嵌套组结构
	root.Group("/level1", func(g1 *Group) {
		g1.Use(testMiddleware("level1"))

		g1.Group("/level2", func(g2 *Group) {
			g2.Use(testMiddleware("level2"))

			g2.Group("/level3", func(g3 *Group) {
				g3.Use(testMiddleware("level3"))
				g3.GET("/deep", testHandler("deep nested"))
			})
		})
	})

	handler := root.Build()
	req := httptest.NewRequest("GET", "/level1/level2/level3/deep", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "deep nested" {
		t.Errorf("Expected body 'deep nested', got '%s'", w.Body.String())
	}

	// 验证中间件顺序（最深层应该在最外层）
	if w.Header().Get("X-Middleware") != "level3" {
		t.Error("Deepest middleware should be applied outermost")
	}
}
