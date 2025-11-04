package middleware

import (
	"net/http"

	"github.com/lyuangg/yuango/internal/app"
	"github.com/lyuangg/yuango/internal/trace"
)

const (
	// TraceIDHeader 是 HTTP 请求头中 trace_id 的字段名
	TraceIDHeader = "X-Trace-ID"
)

// TraceID 中间件：为每个请求添加或复用 trace_id
// 1. 如果请求头中有 X-Trace-ID，则使用该值
// 2. 如果没有，则生成一个新的 trace_id
// 3. 将 trace_id 设置到 context 中
// 4. 将 trace_id 添加到响应头中
func TraceID(appCtx *app.AppContext) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var traceID string

			// 1. 尝试从请求头获取 trace_id
			if headerTraceID := r.Header.Get(TraceIDHeader); headerTraceID != "" {
				traceID = headerTraceID
			} else {
				// 2. 生成新的 trace_id
				traceID = trace.NewTraceID()
			}

			// 3. 设置到 context 中
			ctx := trace.SetTraceID(r.Context(), traceID)

			// 4. 添加到响应头中（便于客户端追踪）
			w.Header().Set(TraceIDHeader, traceID)

			// 5. 传递给下一个处理器
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
