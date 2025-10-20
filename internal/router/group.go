package router

import (
	"net/http"

	"github.com/lyuangg/yuango/internal/app"
	"github.com/lyuangg/yuango/internal/middleware"
)

// Group represents a route group with shared middleware and prefix.
type Group struct {
	prefix      string
	middlewares []middleware.Middleware
	appCtx      *app.AppContext
	mux         *http.ServeMux
}

// NewGroup creates a new route group.
func NewGroup(appCtx *app.AppContext) *Group {
	return &Group{
		appCtx:      appCtx,
		mux:         http.NewServeMux(),
		middlewares: make([]middleware.Middleware, 0),
	}
}

// Use adds middleware to the group.
// Middleware will be applied in the order they are added.
func (g *Group) Use(middlewares ...middleware.Middleware) *Group {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

// GET registers a GET route.
func (g *Group) GET(pattern string, handler http.HandlerFunc) *Group {
	g.mux.HandleFunc("GET "+pattern, handler)
	return g
}

// POST registers a POST route.
func (g *Group) POST(pattern string, handler http.HandlerFunc) *Group {
	g.mux.HandleFunc("POST "+pattern, handler)
	return g
}

// PUT registers a PUT route.
func (g *Group) PUT(pattern string, handler http.HandlerFunc) *Group {
	g.mux.HandleFunc("PUT "+pattern, handler)
	return g
}

// DELETE registers a DELETE route.
func (g *Group) DELETE(pattern string, handler http.HandlerFunc) *Group {
	g.mux.HandleFunc("DELETE "+pattern, handler)
	return g
}

// PATCH registers a PATCH route.
func (g *Group) PATCH(pattern string, handler http.HandlerFunc) *Group {
	g.mux.HandleFunc("PATCH "+pattern, handler)
	return g
}

// Handle registers a route with any HTTP method.
func (g *Group) Handle(pattern string, handler http.Handler) *Group {
	g.mux.Handle(pattern, handler)
	return g
}

// HandleFunc registers a route handler function.
func (g *Group) HandleFunc(pattern string, handler http.HandlerFunc) *Group {
	g.mux.HandleFunc(pattern, handler)
	return g
}

// Group creates a sub-group with additional prefix and middleware.
func (g *Group) Group(prefix string, fn func(*Group)) *Group {
	subGroup := &Group{
		appCtx:      g.appCtx,
		mux:         http.NewServeMux(),
		middlewares: make([]middleware.Middleware, len(g.middlewares)),
	}
	// Copy parent middlewares
	copy(subGroup.middlewares, g.middlewares)

	// Configure sub-group
	fn(subGroup)

	// Mount sub-group with prefix
	handler := subGroup.Build()
	if prefix != "" {
		handler = http.StripPrefix(prefix, handler)
		g.mux.Handle(prefix+"/", handler)
	} else {
		g.mux.Handle("/", handler)
	}

	return g
}

// Build returns the final http.Handler with all middlewares applied.
func (g *Group) Build() http.Handler {
	var handler http.Handler = g.mux

	// Apply middlewares in reverse order (last added = outermost)
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		handler = g.middlewares[i](handler)
	}

	return handler
}
