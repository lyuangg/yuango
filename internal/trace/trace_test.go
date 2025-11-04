package trace

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func TestSetAndGetTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-id-12345"

	// 设置 trace_id
	ctx = SetTraceID(ctx, traceID)

	// 获取 trace_id
	got := GetTraceID(ctx)
	if got != traceID {
		t.Errorf("GetTraceID() = %v, want %v", got, traceID)
	}
}

func TestGetTraceID_Empty(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "nil context",
			ctx:  nil,
			want: "",
		},
		{
			name: "empty context",
			ctx:  context.Background(),
			want: "",
		},
		{
			name: "context with wrong type",
			ctx:  context.WithValue(context.Background(), TraceIDKey, 12345),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTraceID(tt.ctx)
			if got != tt.want {
				t.Errorf("GetTraceID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTraceID(t *testing.T) {
	traceID := NewTraceID()

	// 验证格式: {timestamp}-{random}-{counter}
	parts := strings.Split(traceID, "-")
	if len(parts) != 3 {
		t.Errorf("NewTraceID() format invalid, got %v, expected 3 parts", traceID)
	}

	// 验证时间戳部分（14位数字）
	if len(parts[0]) != 14 {
		t.Errorf("timestamp part length = %d, want 14", len(parts[0]))
	}

	// 验证随机数部分（8位十六进制）
	if len(parts[1]) != 8 {
		t.Errorf("random part length = %d, want 8", len(parts[1]))
	}

	// 验证计数器部分（4位数字）
	if len(parts[2]) != 4 {
		t.Errorf("counter part length = %d, want 4", len(parts[2]))
	}
}

func TestNewTraceID_Uniqueness(t *testing.T) {
	const count = 1000
	ids := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		id := NewTraceID()
		if ids[id] {
			t.Errorf("NewTraceID() generated duplicate: %s", id)
		}
		ids[id] = true
	}
}

func TestNewTraceID_Concurrent(t *testing.T) {
	const goroutines = 100
	const idsPerGoroutine = 100

	var wg sync.WaitGroup
	idsChan := make(chan string, goroutines*idsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				idsChan <- NewTraceID()
			}
		}()
	}

	wg.Wait()
	close(idsChan)

	// 检查唯一性
	ids := make(map[string]bool)
	for id := range idsChan {
		if ids[id] {
			t.Errorf("NewTraceID() generated duplicate in concurrent test: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != goroutines*idsPerGoroutine {
		t.Errorf("Expected %d unique IDs, got %d", goroutines*idsPerGoroutine, len(ids))
	}
}

func TestWithNewTraceID(t *testing.T) {
	ctx := context.Background()

	// 创建带有 trace_id 的 context
	ctx = WithNewTraceID(ctx)

	// 验证 trace_id 存在
	traceID := GetTraceID(ctx)
	if traceID == "" {
		t.Error("WithNewTraceID() should set a trace_id")
	}

	// 验证格式
	parts := strings.Split(traceID, "-")
	if len(parts) != 3 {
		t.Errorf("trace_id format invalid: %s", traceID)
	}
}

func TestGetOrCreateTraceID_Existing(t *testing.T) {
	ctx := context.Background()
	existingID := "existing-trace-id"
	ctx = SetTraceID(ctx, existingID)

	// 应该返回已存在的 trace_id
	newCtx, traceID := GetOrCreateTraceID(ctx)
	if traceID != existingID {
		t.Errorf("GetOrCreateTraceID() = %v, want %v", traceID, existingID)
	}

	// context 应该不变
	if GetTraceID(newCtx) != existingID {
		t.Error("GetOrCreateTraceID() should not modify existing trace_id")
	}
}

func TestGetOrCreateTraceID_New(t *testing.T) {
	ctx := context.Background()

	// 应该创建新的 trace_id
	newCtx, traceID := GetOrCreateTraceID(ctx)
	if traceID == "" {
		t.Error("GetOrCreateTraceID() should create a new trace_id")
	}

	// 验证新 context 包含 trace_id
	if GetTraceID(newCtx) != traceID {
		t.Error("GetOrCreateTraceID() should set trace_id in returned context")
	}

	// 验证格式
	parts := strings.Split(traceID, "-")
	if len(parts) != 3 {
		t.Errorf("trace_id format invalid: %s", traceID)
	}
}

func BenchmarkNewTraceID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTraceID()
	}
}

func BenchmarkSetTraceID(b *testing.B) {
	ctx := context.Background()
	traceID := "test-trace-id"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SetTraceID(ctx, traceID)
	}
}

func BenchmarkGetTraceID(b *testing.B) {
	ctx := SetTraceID(context.Background(), "test-trace-id")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		GetTraceID(ctx)
	}
}
