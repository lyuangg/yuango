package queue

import (
	"context"
	"testing"
	"time"

	"github.com/lyuangg/yuango/internal/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisQueue(t *testing.T) {
	ctx := context.Background()

	t.Run("push_and_pop", func(t *testing.T) {
		mockClient := redis.NewMockClient()
		queue := NewRedisQueue(mockClient)
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
		mockClient := redis.NewMockClient()
		queue := NewRedisQueue(mockClient)
		queueName := "test:empty"

		msg, err := queue.Pop(ctx, queueName)
		assert.Error(t, err)
		assert.Equal(t, ErrQueueEmpty, err)
		assert.Nil(t, msg)
	})

	t.Run("push_delayed", func(t *testing.T) {
		mockClient := redis.NewMockClient()
		queue := NewRedisQueue(mockClient)
		queueName := "test:delayed"

		payload := []byte("delayed-message")

		var msg *Message
		err := queue.PushDelayed(ctx, queueName, payload, 100*time.Millisecond)
		require.NoError(t, err)

		// Should not be available immediately
		_, err = queue.Pop(ctx, queueName)
		assert.Error(t, err)
		assert.Equal(t, ErrQueueEmpty, err)

		// Wait for delay
		time.Sleep(150 * time.Millisecond)

		// Should be available now
		msg, err = queue.Pop(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, payload, msg.Payload)
	})

	t.Run("push_delayed_negative", func(t *testing.T) {
		mockClient := redis.NewMockClient()
		queue := NewRedisQueue(mockClient)
		queueName := "test:delayed_neg"

		payload := []byte("test")
		err := queue.PushDelayed(ctx, queueName, payload, -1*time.Second)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidDelay, err)
	})

	t.Run("size", func(t *testing.T) {
		mockClient := redis.NewMockClient()
		queue := NewRedisQueue(mockClient)
		queueName := "test:size"

		// Push multiple messages
		queue.Push(ctx, queueName, []byte("msg1"))
		queue.Push(ctx, queueName, []byte("msg2"))
		queue.PushDelayed(ctx, queueName, []byte("msg3"), time.Second)

		size, err := queue.Size(ctx, queueName)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, size, int64(3))
	})

	t.Run("pop_with_timeout", func(t *testing.T) {
		mockClient := redis.NewMockClient()
		queue := NewRedisQueue(mockClient)
		queueName := "test:timeout"

		// Push a message first
		payload := []byte("timeout-test")
		err := queue.Push(ctx, queueName, payload)
		require.NoError(t, err)

		// Should get it immediately
		msg, err := queue.PopWithTimeout(ctx, queueName, 100*time.Millisecond)
		require.NoError(t, err)
		assert.Equal(t, payload, msg.Payload)

		// Test with timeout when queue is empty
		msg, err = queue.PopWithTimeout(ctx, queueName, 100*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, ErrQueueEmpty, err)
		assert.Nil(t, msg)
	})

	t.Run("multiple_queues", func(t *testing.T) {
		mockClient := redis.NewMockClient()
		queue := NewRedisQueue(mockClient)
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
}
