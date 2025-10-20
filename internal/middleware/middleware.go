// Package middleware provides HTTP middleware functions using closure pattern.
package middleware

import "net/http"

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler
