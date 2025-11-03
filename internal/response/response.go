// Package response provides utilities for HTTP response handling.
package response

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/lyuangg/yuango/internal/errors"
	"github.com/lyuangg/yuango/internal/trace"
)

// ====================
// JSON Response
// ====================

// JSONResponse is the standard JSON response format.
// Default fields: code, msg, data, trace_id
type JSONResponse struct {
	Code    int         `json:"code"`     // Business status code: 0=success, others=failure
	Message string      `json:"msg"`      // Response message
	Data    interface{} `json:"data"`     // Response data
	TraceID string      `json:"trace_id"` // Trace ID (obtained from context)
}

// NewJSONResponse creates a new JSONResponse.
func NewJSONResponse(ctx context.Context, code int, msg string, data interface{}) JSONResponse {
	return JSONResponse{
		Code:    code,
		Message: msg,
		Data:    data,
		TraceID: trace.GetTraceID(ctx),
	}
}

// Success returns a success response (code=0).
func Success(ctx context.Context, w http.ResponseWriter, data interface{}) error {
	resp := NewJSONResponse(ctx, 0, "success", data)
	return WriteJSON(w, http.StatusOK, resp)
}

// Fail returns a failure response (accepts error parameter).
func Fail(ctx context.Context, w http.ResponseWriter, err error) error {
	// Check if it's an APIError
	apiErr, ok := errors.AsAPIError(err)
	if !ok {
		apiErr = errors.ErrInternalServer.Wrap(err)
	}

	// Create response using APIError information
	resp := NewJSONResponse(ctx, apiErr.Code, apiErr.Message, nil)
	return WriteJSON(w, http.StatusOK, resp)
}

// ====================
// Text Response
// ====================

// Text returns a plain text response.
func Text(w http.ResponseWriter, httpStatus int, text string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(httpStatus)
	_, err := w.Write([]byte(text))
	return err
}

// HTML returns an HTML response.
func HTML(w http.ResponseWriter, httpStatus int, html string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(httpStatus)
	_, err := w.Write([]byte(html))
	return err
}

// NoContent returns a no content response (204).
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// ====================
// Helper Functions
// ====================

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, httpStatus int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)

	return json.NewEncoder(w).Encode(data)
}
