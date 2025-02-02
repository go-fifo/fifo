// Package fifo provides a thread-safe FIFO queue with resizable capacity.
package fifo

import (
	"errors"
	"sync"
)

// ErrQueueFull is returned when an attempt is made to add an element to a full queue.
var ErrQueueFull = errors.New("queue is full")

// ErrQueueEmpty is returned when an attempt is made to remove an element from an empty queue.
var ErrQueueEmpty = errors.New("queue is empty")

// ErrCapacityNotPositive is returned when an attempt is made to create a queue with a non-positive capacity.
var ErrCapacityNotPositive = errors.New("capacity must be positive")

// ErrQueueClosed is returned when an attempt is made to perform an operation on a closed queue.
var ErrQueueClosed = errors.New("queue is closed")

// Queue is a thread-safe FIFO queue with resizable capacity.
type Queue[T any] struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []T
	head   int
	tail   int
	len    int
	cap    int
	closed bool
}

// New creates a new Queue with the given initial capacity, or panics if the capacity is not positive.
func New[T any](initialCapacity int) *Queue[T] {
	if initialCapacity <= 0 {
		panic(ErrCapacityNotPositive)
	}
	q := &Queue[T]{
		items: make([]T, initialCapacity),
		cap:   initialCapacity,
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Len returns the number of items in the queue.
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.len
}

// Cap returns the current capacity of the queue.
func (q *Queue[T]) Cap() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.cap
}

// TryEnqueue attempts to add an item to the end of the queue. If the queue is full, ErrQueueFull is returned immediately.
func (q *Queue[T]) TryEnqueue(item T) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	if q.len >= q.cap {
		return ErrQueueFull
	}

	q.items[q.tail] = item
	q.tail = (q.tail + 1) % cap(q.items)
	q.len++
	q.cond.Broadcast()

	return nil
}

// Enqueue adds an item to the end of the queue. If the queue is full, the calling goroutine is blocked until space becomes available.
func (q *Queue[T]) Enqueue(item T) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.len >= q.cap && !q.closed {
		q.cond.Wait()
	}

	if q.closed {
		return ErrQueueClosed
	}

	q.items[q.tail] = item
	q.tail = (q.tail + 1) % cap(q.items)
	q.len++
	q.cond.Broadcast()

	return nil
}

// TryDequeue attempts to remove and returns the item at the front of the queue. If the queue is empty, ErrQueueEmpty is returned immediately.
func (q *Queue[T]) TryDequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var zero T
	if q.len == 0 {
		if q.closed {
			return zero, ErrQueueClosed
		}
		return zero, ErrQueueEmpty
	}

	item := q.items[q.head]
	q.items[q.head] = zero // Clear the reference to allow garbage collection
	q.head = (q.head + 1) % cap(q.items)
	q.len--
	q.cond.Broadcast()

	return item, nil
}

// Dequeue removes and returns the item at the front of the queue. If the queue is empty, the calling goroutine is blocked until an item becomes available.
func (q *Queue[T]) Dequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var zero T
	for q.len == 0 {
		if q.closed {
			return zero, ErrQueueClosed
		}
		q.cond.Wait()
	}

	item := q.items[q.head]
	q.items[q.head] = zero
	q.head = (q.head + 1) % cap(q.items)
	q.len--
	q.cond.Broadcast()

	return item, nil
}

// Resize changes the capacity of the queue. It returns an error if the new capacity is not positive, or if the queue is closed.
func (q *Queue[T]) Resize(newCap int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if newCap == q.cap {
		return nil
	}
	if newCap <= 0 {
		return ErrCapacityNotPositive
	}
	if q.closed {
		return ErrQueueClosed
	}

	// Ensure no data loss
	ns := newCap
	if q.len > ns {
		ns = q.len
	}
	newItems := make([]T, ns)

	// Copy data from old slice to new slice
	if q.len > 0 {
		if q.head < q.tail {
			copy(newItems, q.items[q.head:q.tail])
		} else {
			n := copy(newItems, q.items[q.head:])
			copy(newItems[n:], q.items[:q.tail])
		}
	}

	q.items = newItems
	q.head = 0
	q.tail = q.len % ns // Adjust the tail position based on the actual capacity of the new slice
	q.cap = newCap
	q.cond.Broadcast() // Wake up all goroutines waiting due to full queue

	return nil
}

// Close marks the queue as closed, preventing further enqueues.
func (q *Queue[T]) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	q.closed = true
	q.cond.Broadcast()
	return nil
}
