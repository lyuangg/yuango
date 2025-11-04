package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lyuangg/yuango/internal/trace"
)

func TestTraceID_WithExistingTraceID(t *testing.T) {
	existingTraceID := "existing-trace-id-12345"
	var capturedTraceID string

	// 创建测试 handler
	handler := TraceID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID = trace.GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 创建带有 X-Trace-ID 的请求
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TraceIDHeader, existingTraceID)
	w := httptest.NewRecorder()

	// 执行请求
	handler.ServeHTTP(w, req)

	// 验证 context 中的 trace_id
	if capturedTraceID != existingTraceID {
		t.Errorf("context trace_id = %v, want %v", capturedTraceID, existingTraceID)
	}

	// 验证响应头中的 trace_id
	if w.Header().Get(TraceIDHeader) != existingTraceID {
		t.Errorf("response header trace_id = %v, want %v", w.Header().Get(TraceIDHeader), existingTraceID)
	}
}

func TestTraceID_WithoutExistingTraceID(t *testing.T) {
	var capturedTraceID string

	// 创建测试 handler
	handler := TraceID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID = trace.GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 创建不带 X-Trace-ID 的请求
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// 执行请求
	handler.ServeHTTP(w, req)

	// 验证生成了新的 trace_id
	if capturedTraceID == "" {
		t.Error("should generate a new trace_id")
	}

	// 验证响应头中的 trace_id
	responseTraceID := w.Header().Get(TraceIDHeader)
	if responseTraceID == "" {
		t.Error("response header should contain trace_id")
	}

	// 验证 context 和响应头中的 trace_id 一致
	if capturedTraceID != responseTraceID {
		t.Errorf("context trace_id (%v) != response header trace_id (%v)", capturedTraceID, responseTraceID)
	}
}

func TestTraceID_Propagation(t *testing.T) {
	var innerTraceID string

	// 创建嵌套 handler
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerTraceID = trace.GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// 包装两层 middleware
	handler := TraceID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 在这里也能获取到 trace_id
		outerTraceID := trace.GetTraceID(r.Context())

		// 调用内层 handler
		innerHandler.ServeHTTP(w, r)

		// 验证 trace_id 在整个链路中保持一致
		if outerTraceID != innerTraceID {
			t.Errorf("trace_id not propagated correctly: outer=%v, inner=%v", outerTraceID, innerTraceID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func TestTraceID_EmptyHeader(t *testing.T) {
	var capturedTraceID string

	handler := TraceID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID = trace.GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 设置空的 X-Trace-ID 头
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TraceIDHeader, "")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// 应该生成新的 trace_id（因为头是空的）
	if capturedTraceID == "" {
		t.Error("should generate a new trace_id when header is empty")
	}
}

func TestTraceID_ConcurrentRequests(t *testing.T) {
	handler := TraceID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := trace.GetTraceID(r.Context())
		if traceID == "" {
			t.Error("trace_id should not be empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	// 并发发送多个请求
	const numRequests = 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			done <- true
		}()
	}

	// 等待所有请求完成
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

func BenchmarkTraceID_WithHeader(b *testing.B) {
	handler := TraceID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TraceIDHeader, "existing-trace-id")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkTraceID_WithoutHeader(b *testing.B) {
	handler := TraceID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
