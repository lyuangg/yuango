package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryQueue(t *testing.T) {
	ctx := context.Background()

	t.Run("push_and_pop", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:push_pop"

		payload := []byte("test-message")

		err := queue.Push(ctx, queueName, payload)
		require.NoError(t, err)

		msg, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, payload, msg.Payload)
		assert.NotEmpty(t, msg.ID)
	})

	t.Run("pop_empty_queue", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:empty"

		msg, err := queue.Pop(ctx, queueName)
		assert.Error(t, err)
		assert.Equal(t, ErrQueueEmpty, err)
		assert.Nil(t, msg)
	})

	t.Run("push_delayed", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:delayed"

		payload := []byte("delayed-message")

		err := queue.PushDelayed(ctx, queueName, payload, 100*time.Millisecond)
		require.NoError(t, err)

		// Should not be available immediately
		_, err = queue.Pop(ctx, queueName)
		assert.Error(t, err)
		assert.Equal(t, ErrQueueEmpty, err)

		// Wait for delay
		time.Sleep(150 * time.Millisecond)

		// Should be available now
		msg, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, payload, msg.Payload)
	})

	t.Run("push_delayed_negative", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:delayed_neg"

		payload := []byte("test")
		err := queue.PushDelayed(ctx, queueName, payload, -1*time.Second)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidDelay, err)
	})

	t.Run("size", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:size"

		// Push multiple messages
		queue.Push(ctx, queueName, []byte("msg1"))
		queue.Push(ctx, queueName, []byte("msg2"))
		queue.PushDelayed(ctx, queueName, []byte("msg3"), time.Second)

		size, err := queue.Size(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, int64(3), size)
	})

	t.Run("pop_with_timeout", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:timeout"

		// Push a message first
		payload := []byte("timeout-test")
		err := queue.Push(ctx, queueName, payload)
		require.NoError(t, err)

		// Should get it immediately (non-blocking when timeout=0)
		msg, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, payload, msg.Payload)

		// Test with timeout when queue is empty (will wait and timeout)
		msg, err = queue.PopWithTimeout(ctx, queueName, 50*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, ErrQueueEmpty, err)
		assert.Nil(t, msg)
	})

	t.Run("multiple_queues", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queue1 := "test:queue1"
		queue2 := "test:queue2"

		queue.Push(ctx, queue1, []byte("msg1"))
		queue.Push(ctx, queue2, []byte("msg2"))

		msg1, err := queue.Pop(ctx, queue1)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg1"), msg1.Payload)

		msg2, err := queue.Pop(ctx, queue2)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg2"), msg2.Payload)
	})

	t.Run("fifo_order", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:fifo"

		// Push multiple messages
		queue.Push(ctx, queueName, []byte("msg1"))
		queue.Push(ctx, queueName, []byte("msg2"))
		queue.Push(ctx, queueName, []byte("msg3"))

		// Pop should return in FIFO order
		msg1, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg1"), msg1.Payload)

		msg2, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg2"), msg2.Payload)

		msg3, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg3"), msg3.Payload)
	})

	t.Run("delayed_messages_order", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:delayed_order"

		// Push delayed messages with different delays
		queue.PushDelayed(ctx, queueName, []byte("msg3"), 300*time.Millisecond)
		queue.PushDelayed(ctx, queueName, []byte("msg1"), 100*time.Millisecond)
		queue.PushDelayed(ctx, queueName, []byte("msg2"), 200*time.Millisecond)

		// Wait for first message
		time.Sleep(150 * time.Millisecond)
		msg1, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg1"), msg1.Payload)

		// Wait for second message
		time.Sleep(100 * time.Millisecond)
		msg2, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg2"), msg2.Payload)

		// Wait for third message
		time.Sleep(100 * time.Millisecond)
		msg3, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg3"), msg3.Payload)
	})

	t.Run("close", func(t *testing.T) {
		queue := NewMemoryQueue()
		queueName := "test:close"

		// Push a message
		err := queue.Push(ctx, queueName, []byte("msg"))
		require.NoError(t, err)

		// Close queue
		err = queue.Close()
		assert.NoError(t, err)

		// Try to push after close
		err = queue.Push(ctx, queueName, []byte("msg2"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed")

		// Try to pop after close (should still work for existing messages)
		msg, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("msg"), msg.Payload)

		// Try to get size after close
		size, err := queue.Size(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, int64(0), size)
	})

	t.Run("size_after_pop", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:size_pop"

		queue.Push(ctx, queueName, []byte("msg1"))
		queue.Push(ctx, queueName, []byte("msg2"))

		size, err := queue.Size(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, int64(2), size)

		queue.Pop(ctx, queueName)

		size, err = queue.Size(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, int64(1), size)
	})

	t.Run("mixed_immediate_and_delayed", func(t *testing.T) {
		queue := NewMemoryQueue()
		defer queue.Close()
		queueName := "test:mixed"

		// Push immediate message
		queue.Push(ctx, queueName, []byte("immediate"))

		// Push delayed message
		queue.PushDelayed(ctx, queueName, []byte("delayed"), 100*time.Millisecond)

		// Should get immediate message first
		msg, err := queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("immediate"), msg.Payload)

		// Delayed message not ready yet
		_, err = queue.Pop(ctx, queueName)
		assert.Error(t, err)
		assert.Equal(t, ErrQueueEmpty, err)

		// Wait for delayed message
		time.Sleep(150 * time.Millisecond)
		msg, err = queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, []byte("delayed"), msg.Payload)
	})
}

// TestMemoryQueue_Concurrent tests concurrent access to MemoryQueue.
func TestMemoryQueue_Concurrent(t *testing.T) {
	queue := NewMemoryQueue()
	defer queue.Close()
	ctx := context.Background()
	queueName := "test:concurrent"

	t.Run("concurrent_push_and_pop", func(t *testing.T) {
		const numGoroutines = 50
		const messagesPerGoroutine = 10

		var wg sync.WaitGroup
		errorCount := int64(0)

		// Concurrent pushes
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					payload := []byte(fmt.Sprintf("msg:%d:%d", goroutineID, j))
					if err := queue.Push(ctx, queueName, payload); err != nil {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		assert.Equal(t, int64(0), errorCount, "No errors should occur during concurrent pushes")

		// Verify all messages are in queue
		size, err := queue.Size(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, int64(numGoroutines*messagesPerGoroutine), size)

		// Concurrent pops
		popCount := int64(0)
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					msg, err := queue.Pop(ctx, queueName)
					if err == nil && msg != nil {
						atomic.AddInt64(&popCount, 1)
					}
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, int64(numGoroutines*messagesPerGoroutine), popCount, "All messages should be popped")
	})

	t.Run("concurrent_delayed_messages", func(t *testing.T) {
		const numGoroutines = 20
		queueName := "test:concurrent_delayed"

		var wg sync.WaitGroup

		// Concurrent delayed pushes
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				delay := time.Duration(id+1) * 50 * time.Millisecond
				payload := []byte(fmt.Sprintf("delayed:%d", id))
				queue.PushDelayed(ctx, queueName, payload, delay)
			}(i)
		}

		wg.Wait()

		// Wait for all messages to be ready
		time.Sleep(time.Duration(numGoroutines+1) * 50 * time.Millisecond)

		// Verify all messages are ready
		size, err := queue.Size(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, int64(numGoroutines), size)

		// Pop all messages
		for i := 0; i < numGoroutines; i++ {
			msg, err := queue.Pop(ctx, queueName)
			require.NoError(t, err)
			assert.NotNil(t, msg)
		}
	})
}
