package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/lyuangg/yuango/internal/app"
	"github.com/lyuangg/yuango/internal/response"
)

// Recovery returns a middleware that recovers from panics and logs stack trace.
func Recovery(appCtx *app.AppContext) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Capture full stack trace
					stack := debug.Stack()

					if appCtx != nil && appCtx.Logger != nil {
						appCtx.Logger.Error(r.Context(), "panic recovered",
							"error", fmt.Sprintf("%v", err),
							"method", r.Method,
							"path", r.URL.Path,
							"remote_addr", r.RemoteAddr,
							"user_agent", r.UserAgent(),
							"stack", string(stack),
						)
					}

					// Convert panic value to error
					var panicErr error
					if e, ok := err.(error); ok {
						panicErr = e
					} else {
						panicErr = fmt.Errorf("%v", err)
					}

					if writeErr := response.Fail(r.Context(), w, panicErr); writeErr != nil {
						appCtx.Logger.Error(r.Context(), "failed to write error response", "error", writeErr)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
