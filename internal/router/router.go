// Package router provides HTTP routing configuration for the application.
package router

import (
	"net/http"

	"github.com/lyuangg/yuango/internal/app"
	"github.com/lyuangg/yuango/internal/handler"
	"github.com/lyuangg/yuango/internal/middleware"
)

// New builds the application's http.Handler with routes and middleware.
func New(appCtx *app.AppContext) http.Handler {
	// 创建根路由组
	root := NewGroup(appCtx)

	// 全局中间件（应用到所有路由）
	root.Use(middleware.Recovery(appCtx))

	// ====================
	// 公开路由（无额外中间件）
	// ====================
	root.GET("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	root.GET("/ping", handler.Ping(appCtx))

	// ====================
	// API 路由组（/api/*）
	// ====================
	root.Group("/api", func(g *Group) {
		// API 路由组中间件
		g.Use(middleware.Logging(appCtx))

		// 公开 API 路由
		g.GET("/status", handler.Ping(appCtx))

		// 用户相关路由
		g.GET("/users", handler.Ping(appCtx))
		g.GET("/users/{id}", handler.Ping(appCtx))
		g.POST("/users", handler.Ping(appCtx))
		g.PUT("/users/{id}", handler.Ping(appCtx))
		g.DELETE("/users/{id}", handler.Ping(appCtx))

		// 需要认证的路由子组
		g.Group("/protected", func(sg *Group) {
			sg.Use(middleware.Auth(appCtx))
			sg.GET("/profile", handler.Ping(appCtx))
			sg.GET("/settings", handler.Ping(appCtx))
		})
	})

	// ====================
	// Admin 路由组（/admin/*）
	// ====================
	root.Group("/admin", func(g *Group) {
		// Admin 路由组中间件（日志 + 认证）
		g.Use(
			middleware.Logging(appCtx),
			middleware.Auth(appCtx),
		)

		g.GET("/dashboard", handler.Ping(appCtx))
		g.GET("/settings", handler.Ping(appCtx))
		g.GET("/users", handler.Ping(appCtx))

		// Admin 统计子组（可以添加额外的权限检查）
		g.Group("/stats", func(sg *Group) {
			// sg.Use(middleware.AdminOnly(appCtx)) // 可以添加额外的中间件
			sg.GET("/overview", handler.Ping(appCtx))
			sg.GET("/reports", handler.Ping(appCtx))
		})
	})

	return root.Build()
}
