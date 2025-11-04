// Package trace provides distributed tracing ID management for request tracking.
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

// contextKey 用于存储 context 中的键
type contextKey string

const (
	// TraceIDKey 是 trace_id 在 context 中的键
	TraceIDKey contextKey = "trace_id"
)

// 全局计数器，用于生成唯一的序列号
var counter uint64

// SetTraceID 将 trace_id 设置到 context 中
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从 context 中获取 trace_id
// 如果 context 中没有 trace_id，返回空字符串
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// NewTraceID 生成一个新的 trace_id
// 格式: {timestamp}-{random}-{counter}
// 例如: 20231020-a1b2c3d4-0001
func NewTraceID() string {
	// 1. 时间戳部分（紧凑格式）
	timestamp := time.Now().Format("20060102150405")

	// 2. 随机数部分（4 字节）
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		// 如果随机数生成失败，使用时间戳的纳秒部分
		randomBytes = []byte(fmt.Sprintf("%04d", time.Now().Nanosecond()%10000))
	}
	randomHex := hex.EncodeToString(randomBytes)

	// 3. 计数器部分（原子递增）
	seq := atomic.AddUint64(&counter, 1)

	// 组合成最终的 trace_id
	return fmt.Sprintf("%s-%s-%04d", timestamp, randomHex, seq%10000)
}

// WithNewTraceID 创建一个带有新 trace_id 的 context
func WithNewTraceID(ctx context.Context) context.Context {
	return SetTraceID(ctx, NewTraceID())
}

// GetOrCreateTraceID 从 context 获取 trace_id，如果不存在则创建一个新的
func GetOrCreateTraceID(ctx context.Context) (context.Context, string) {
	if traceID := GetTraceID(ctx); traceID != "" {
		return ctx, traceID
	}

	traceID := NewTraceID()
	return SetTraceID(ctx, traceID), traceID
}
