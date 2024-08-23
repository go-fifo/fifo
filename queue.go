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

// ErrNewCapacityTooSmall is returned when an attempt is made to resize the queue to a capacity smaller than the current number of items.
var ErrNewCapacityTooSmall = errors.New("new capacity is too small")

// Queue represents a thread-safe FIFO queue with resizable capacity.
type Queue[T any] struct {
	gMu   sync.RWMutex
	rsMu  sync.Mutex
	items []T
	head  int
	tail  int
	len   int
	cap   int
}

// New creates a new Queue with the given initial capacity.
func New[T any](initialCapacity int) *Queue[T] {
	return &Queue[T]{
		items: make([]T, initialCapacity),
		cap:   initialCapacity,
	}
}

// Enqueue adds an item to the queue. It returns an error if the queue is full.
func (q *Queue[T]) Enqueue(item T) error {
	q.gMu.Lock()
	defer q.gMu.Unlock()

	if q.len == q.cap {
		return ErrQueueFull
	}

	q.items[q.tail] = item
	q.tail = (q.tail + 1) % q.cap
	q.len++

	return nil
}

// Dequeue removes and returns an item from the queue. It returns an error if the queue is empty.
func (q *Queue[T]) Dequeue() (T, error) {
	q.gMu.Lock()
	defer q.gMu.Unlock()

	var zero T
	if q.len == 0 {
		return zero, ErrQueueEmpty
	}

	item := q.items[q.head]
	q.items[q.head] = zero // Clear the reference to allow garbage collection
	q.head = (q.head + 1) % q.cap
	q.len--

	return item, nil
}

// Len returns the current number of items in the queue.
func (q *Queue[T]) Len() int {
	q.gMu.RLock()
	defer q.gMu.RUnlock()
	return q.len
}

// Cap returns the current capacity of the queue.
func (q *Queue[T]) Cap() int {
	q.gMu.RLock()
	defer q.gMu.RUnlock()
	return q.cap
}

// Resize changes the capacity of the queue. It returns an error if the new capacity is smaller than the current number of items.
func (q *Queue[T]) Resize(newCap int) error {
	q.rsMu.Lock()
	defer q.rsMu.Unlock()

	q.gMu.Lock()
	defer q.gMu.Unlock()

	if newCap == q.cap {
		return nil
	}

	if newCap < q.len {
		return ErrNewCapacityTooSmall
	}

	newItems := make([]T, newCap)
	if q.len > 0 {
		if q.tail > q.head {
			copy(newItems, q.items[q.head:q.tail])
		} else {
			n := copy(newItems, q.items[q.head:])
			copy(newItems[n:], q.items[:q.tail])
		}
	}

	q.items = newItems
	q.head = 0
	q.tail = q.len
	q.cap = newCap

	return nil
}
