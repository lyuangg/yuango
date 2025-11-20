// Package queue provides a queue interface and implementations.
package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryQueue is an in-memory implementation of the Queue interface.
// It can be used as a default queue implementation when Redis is not available,
// or for testing purposes.
type MemoryQueue struct {
	mu       sync.RWMutex
	queues   map[string]*queueData
	closed   bool
	stopChan chan struct{}
}

type queueData struct {
	immediate []Message
	delayed   []delayedMessageItem
}

type delayedMessageItem struct {
	message   Message
	executeAt time.Time
}

// NewMemoryQueue creates a new in-memory queue.
func NewMemoryQueue() *MemoryQueue {
	mq := &MemoryQueue{
		queues:   make(map[string]*queueData),
		stopChan: make(chan struct{}),
	}
	return mq
}

// Push adds a message to the queue immediately.
func (m *MemoryQueue) Push(ctx context.Context, queueName string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("queue is closed")
	}

	queue := m.getOrCreateQueue(queueName)
	msg := Message{
		ID:      generateMessageID(),
		Payload: payload,
	}
	queue.immediate = append(queue.immediate, msg)
	return nil
}

// PushDelayed adds a message to the queue with a delay.
func (m *MemoryQueue) PushDelayed(ctx context.Context, queueName string, payload []byte, delay time.Duration) error {
	if delay < 0 {
		return ErrInvalidDelay
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("queue is closed")
	}

	queue := m.getOrCreateQueue(queueName)
	executeAt := time.Now().Add(delay)
	msg := Message{
		ID:      generateMessageID(),
		Payload: payload,
		Delay:   executeAt,
	}

	// Insert delayed message in sorted order (by executeAt)
	item := delayedMessageItem{
		message:   msg,
		executeAt: executeAt,
	}

	// Find insertion position (keep sorted by executeAt)
	insertPos := 0
	for i, delayed := range queue.delayed {
		if delayed.executeAt.After(executeAt) {
			insertPos = i
			break
		}
		insertPos = i + 1
	}

	// Insert at the correct position
	if insertPos == len(queue.delayed) {
		queue.delayed = append(queue.delayed, item)
	} else {
		queue.delayed = append(queue.delayed[:insertPos+1], queue.delayed[insertPos:]...)
		queue.delayed[insertPos] = item
	}

	return nil
}

// Pop removes and returns a message from the queue.
func (m *MemoryQueue) Pop(ctx context.Context, queueName string) (*Message, error) {
	return m.PopWithTimeout(ctx, queueName, 0)
}

// PopWithTimeout removes and returns a message from the queue with a timeout.
func (m *MemoryQueue) PopWithTimeout(ctx context.Context, queueName string, timeout time.Duration) (*Message, error) {
	// Move ready delayed messages first
	if err := m.moveDelayedMessages(queueName); err != nil {
		return nil, err
	}

	deadline := time.Now()
	if timeout > 0 {
		deadline = deadline.Add(timeout)
	}

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Try to pop a message
		m.mu.Lock()
		queue, exists := m.queues[queueName]
		if !exists || len(queue.immediate) == 0 {
			m.mu.Unlock()

			// If timeout is 0, return immediately
			if timeout == 0 {
				return nil, ErrQueueEmpty
			}

			// Check if timeout expired
			if time.Now().After(deadline) {
				return nil, ErrQueueEmpty
			}

			// Move delayed messages again (in case some became ready)
			if err := m.moveDelayedMessages(queueName); err != nil {
				return nil, err
			}

			// Wait a bit before retrying
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(50 * time.Millisecond):
				continue
			case <-m.stopChan:
				return nil, fmt.Errorf("queue is closed")
			}
		}

		// Pop from immediate queue
		msg := queue.immediate[0]
		queue.immediate = queue.immediate[1:]

		// Clean up empty queue
		if len(queue.immediate) == 0 && len(queue.delayed) == 0 {
			delete(m.queues, queueName)
		}

		m.mu.Unlock()
		return &msg, nil
	}
}

// Size returns the number of messages in the queue.
func (m *MemoryQueue) Size(ctx context.Context, queueName string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	queue, exists := m.queues[queueName]
	if !exists {
		return 0, nil
	}

	return int64(len(queue.immediate) + len(queue.delayed)), nil
}

// Close closes the queue and releases resources.
func (m *MemoryQueue) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	close(m.stopChan)
	return nil
}

// getOrCreateQueue gets or creates a queue data structure.
func (m *MemoryQueue) getOrCreateQueue(queueName string) *queueData {
	if queue, exists := m.queues[queueName]; exists {
		return queue
	}
	queue := &queueData{
		immediate: make([]Message, 0),
		delayed:   make([]delayedMessageItem, 0),
	}
	m.queues[queueName] = queue
	return queue
}

// moveDelayedMessages moves ready delayed messages to the immediate queue.
func (m *MemoryQueue) moveDelayedMessages(queueName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	queue, exists := m.queues[queueName]
	if !exists {
		return nil
	}

	now := time.Now()
	var readyCount int
	for i, delayed := range queue.delayed {
		if delayed.executeAt.After(now) {
			// All remaining messages are not ready yet (sorted order)
			break
		}
		readyCount = i + 1
	}

	if readyCount == 0 {
		return nil
	}

	// Move ready messages to immediate queue
	for i := 0; i < readyCount; i++ {
		queue.immediate = append(queue.immediate, queue.delayed[i].message)
	}

	// Remove moved messages from delayed queue
	queue.delayed = queue.delayed[readyCount:]

	return nil
}
