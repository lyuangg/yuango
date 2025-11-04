package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lyuangg/yuango/internal/app"
)

const (
	// maxBodySize 是记录请求/响应体的最大大小（1MB）
	maxBodySize = 1024 * 1024
)

// isJSONContentType 检查 Content-Type 是否是 JSON 类型
func isJSONContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "application/json")
}

// isValidJSON 快速验证字节数组是否是有效的 JSON
func isValidJSON(data []byte) bool {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return false
	}
	// 快速检查：JSON 必须以 { 或 [ 开头
	if data[0] != '{' && data[0] != '[' {
		return false
	}
	// 完整验证
	var js interface{}
	return json.Unmarshal(data, &js) == nil
}

// responseWriter 包装 http.ResponseWriter 以捕获响应体
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// 同时写入原始 ResponseWriter 和缓冲区
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Flush 实现 http.Flusher 接口
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// readRequestBody 读取请求体（限制大小）
func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	// 读取原始 body
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		return nil, err
	}

	// 重新设置 body，因为 body 只能读取一次
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, nil
}

// getClientIP 获取真实的客户端 IP 地址
// 优先级：X-Forwarded-For > X-Real-IP > RemoteAddr
func getClientIP(r *http.Request) string {
	// 1. 检查 X-Forwarded-For（可能是多个 IP，取第一个）
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		if idx := strings.IndexByte(forwardedFor, ','); idx != -1 {
			if ip := strings.TrimSpace(forwardedFor[:idx]); ip != "" {
				return ip
			}
		} else if ip := strings.TrimSpace(forwardedFor); ip != "" {
			return ip
		}
	}

	// 2. 检查 X-Real-IP
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// 3. 回退到 RemoteAddr（去掉端口号）
	if idx := strings.LastIndexByte(r.RemoteAddr, ':'); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// Logging returns a middleware that logs request information, parameters, and response.
// It only logs JSON format data and skips logging for file uploads/downloads.
func Logging(appCtx *app.AppContext) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 检查是否需要记录日志
			if appCtx == nil || appCtx.Logger == nil {
				next.ServeHTTP(w, r)
				return
			}

			// 记录请求体（只记录 JSON 格式）
			var requestBody string
			if r.Body != nil && isJSONContentType(r.Header.Get("Content-Type")) {
				if bodyBytes, err := readRequestBody(r); err == nil && len(bodyBytes) > 0 && isValidJSON(bodyBytes) {
					requestBody = string(bodyBytes)
				}
			}

			// 包装 ResponseWriter 以捕获响应
			wrapped := newResponseWriter(w)
			next.ServeHTTP(wrapped, r)
			duration := time.Since(start)

			// 记录响应体（只记录 JSON 格式）
			var responseBody string
			if wrapped.body.Len() > 0 && isJSONContentType(wrapped.Header().Get("Content-Type")) {
				bodyBytes := wrapped.body.Bytes()
				if len(bodyBytes) <= maxBodySize {
					if isValidJSON(bodyBytes) {
						responseBody = string(bodyBytes)
					}
				} else if isValidJSON(bodyBytes[:maxBodySize]) {
					responseBody = "<body too large to log>"
				}
			}

			// 构建日志参数
			logArgs := []interface{}{
				"remote_addr", r.RemoteAddr,
				"method", r.Method,
				"path", r.URL.Path,
				"status_code", wrapped.statusCode,
				"duration", duration,
			}

			if clientIP := getClientIP(r); clientIP != "" {
				logArgs = append(logArgs, "client_ip", clientIP)
			}
			if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
				logArgs = append(logArgs, "user_agent", userAgent)
			}
			if queryParams := r.URL.Query(); len(queryParams) > 0 {
				logArgs = append(logArgs, "query_params", queryParams)
			}
			if requestBody != "" {
				logArgs = append(logArgs, "request_body", requestBody)
			}
			if responseBody != "" {
				logArgs = append(logArgs, "response_body", responseBody)
			}

			appCtx.Logger.Info(r.Context(), "request", logArgs...)
		})
	}
}
