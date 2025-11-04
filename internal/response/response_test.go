package response

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lyuangg/yuango/internal/errors"
	"github.com/lyuangg/yuango/internal/trace"
)

// ====================
// NewJSONResponse 测试
// ====================

func TestNewJSONResponse(t *testing.T) {
	tests := []struct {
		name string
		code int
		msg  string
		data interface{}
		want JSONResponse
	}{
		{
			name: "success response",
			code: 0,
			msg:  "success",
			data: map[string]string{"key": "value"},
			want: JSONResponse{
				Code:    0,
				Message: "success",
				Data:    map[string]string{"key": "value"},
				TraceID: "test-trace-123",
			},
		},
		{
			name: "error response",
			code: 500,
			msg:  "internal error",
			data: nil,
			want: JSONResponse{
				Code:    500,
				Message: "internal error",
				Data:    nil,
				TraceID: "test-trace-123",
			},
		},
		{
			name: "empty trace id",
			code: 0,
			msg:  "success",
			data: nil,
			want: JSONResponse{
				Code:    0,
				Message: "success",
				Data:    nil,
				TraceID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name != "empty trace id" {
				ctx = trace.SetTraceID(ctx, "test-trace-123")
			}

			got := NewJSONResponse(ctx, tt.code, tt.msg, tt.data)

			if got.Code != tt.want.Code {
				t.Errorf("Code = %d, want %d", got.Code, tt.want.Code)
			}

			if got.Message != tt.want.Message {
				t.Errorf("Message = %s, want %s", got.Message, tt.want.Message)
			}

			if got.TraceID != tt.want.TraceID {
				t.Errorf("TraceID = %s, want %s", got.TraceID, tt.want.TraceID)
			}
		})
	}
}

// ====================
// Success 测试
// ====================

func TestSuccess(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "with data",
			data: map[string]string{"key": "value"},
		},
		{
			name: "with nil data",
			data: nil,
		},
		{
			name: "with struct",
			data: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{Name: "test", Age: 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx := context.Background()
			ctx = trace.SetTraceID(ctx, "test-trace")

			Success(ctx, w, tt.data)

			// 检查 HTTP 状态码
			if w.Code != http.StatusOK {
				t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
			}

			// 检查 Content-Type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %s, want application/json; charset=utf-8", contentType)
			}

			// 解析响应
			var resp JSONResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// 检查响应结构
			if resp.Code != 0 {
				t.Errorf("Code = %d, want 0", resp.Code)
			}

			if resp.Message != "success" {
				t.Errorf("Message = %s, want success", resp.Message)
			}

			// 验证 TraceID 从 context 中正确获取
			if resp.TraceID != "test-trace" {
				t.Errorf("TraceID = %s, want test-trace", resp.TraceID)
			}
		})
	}
}

// ====================
// Fail 测试
// ====================

func TestFail(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedCode    int
		expectedMessage string
	}{
		{
			name:            "APIError with Details",
			err:             errors.NewWithDetails(500, "Internal Server Error", "database connection failed"),
			expectedCode:    500,
			expectedMessage: "Internal Server Error",
		},
		{
			name:            "APIError without Details",
			err:             errors.New(404, "Not Found"),
			expectedCode:    404,
			expectedMessage: "Not Found",
		},
		{
			name:            "standard Go error",
			err:             stderrors.New("standard Go error"), // 标准 Go error，会被包装为 ErrInternalServer
			expectedCode:    500,                                // 会被包装为 ErrInternalServer
			expectedMessage: "Internal Server Error",
		},
		{
			name:            "nil error",
			err:             nil,
			expectedCode:    500,
			expectedMessage: "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx := context.Background()
			ctx = trace.SetTraceID(ctx, "test-trace")

			Fail(ctx, w, tt.err)

			// 检查 HTTP 状态码
			if w.Code != http.StatusOK {
				t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
			}

			// 检查 Content-Type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %s, want application/json; charset=utf-8", contentType)
			}

			// 解析响应
			var resp JSONResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// 检查响应结构
			if resp.Code != tt.expectedCode {
				t.Errorf("Code = %d, want %d", resp.Code, tt.expectedCode)
			}

			if resp.Message != tt.expectedMessage {
				t.Errorf("Message = %s, want %s", resp.Message, tt.expectedMessage)
			}

			// Fail 函数总是返回 nil data
			if resp.Data != nil {
				t.Errorf("Data = %v, want nil", resp.Data)
			}

			// 验证 TraceID 从 context 中正确获取
			if resp.TraceID != "test-trace" {
				t.Errorf("TraceID = %s, want test-trace", resp.TraceID)
			}
		})
	}
}

// ====================
// Text/HTML/NoContent 测试
// ====================

func TestText(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		text       string
	}{
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			text:       "Hello, World!",
		},
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			text:       "Bad Request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Text(w, tt.statusCode, tt.text)

			// 检查状态码
			if w.Code != tt.statusCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.statusCode)
			}

			// 检查 Content-Type
			contentType := w.Header().Get("Content-Type")
			if contentType != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %s, want text/plain; charset=utf-8", contentType)
			}

			// 检查内容
			if w.Body.String() != tt.text {
				t.Errorf("Body = %s, want %s", w.Body.String(), tt.text)
			}
		})
	}
}

func TestHTML(t *testing.T) {
	htmlContent := "<html><body><h1>Hello</h1></body></html>"

	w := httptest.NewRecorder()
	HTML(w, http.StatusOK, htmlContent)

	// 检查状态码
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	// 检查 Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %s, want text/html; charset=utf-8", contentType)
	}

	// 检查内容
	if w.Body.String() != htmlContent {
		t.Errorf("Body = %s, want %s", w.Body.String(), htmlContent)
	}
}

func TestNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	NoContent(w)

	// 检查状态码
	if w.Code != http.StatusNoContent {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusNoContent)
	}

	// 检查内容为空
	if w.Body.Len() != 0 {
		t.Errorf("Body length = %d, want 0", w.Body.Len())
	}
}

// ====================
// WriteJSON 测试
// ====================

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       interface{}
		wantCode   int
	}{
		{
			name:       "200 OK with map",
			statusCode: http.StatusOK,
			data:       map[string]string{"key": "value"},
			wantCode:   http.StatusOK,
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			data:       map[string]string{"error": "not found"},
			wantCode:   http.StatusNotFound,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			data:       map[string]string{"error": "internal error"},
			wantCode:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSON(w, tt.statusCode, tt.data)

			// 检查状态码
			if w.Code != tt.wantCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantCode)
			}

			// 检查 Content-Type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %s, want application/json; charset=utf-8", contentType)
			}

			// 验证 JSON 格式
			var result map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
				t.Errorf("Failed to decode JSON: %v", err)
			}
		})
	}
}

// ====================
// 并发测试
// ====================

func TestConcurrentWriteJSON(t *testing.T) {
	// 并发写入 JSON 响应
	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			w := httptest.NewRecorder()
			data := map[string]int{"id": id}
			WriteJSON(w, http.StatusOK, data)

			if w.Code != http.StatusOK {
				t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
			}
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
