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
	muEnq sync.Mutex
	muDeq sync.Mutex
	cond  *sync.Cond
	items []T
	head  int
	tail  int
	len   int
	cap   int
}

// New creates a new Queue with the given initial capacity.
func New[T any](initialCapacity int) *Queue[T] {
	q := &Queue[T]{
		items: make([]T, initialCapacity),
		cap:   initialCapacity,
	}
	q.cond = sync.NewCond(&sync.Mutex{})
	return q
}

// Len returns the current number of items in the queue.
func (q *Queue[T]) Len() int {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.len
}

// Cap returns the current capacity of the queue.
func (q *Queue[T]) Cap() int {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.cap
}

// Enqueue adds an item to the queue. It returns an error if the queue is full.
func (q *Queue[T]) Enqueue(item T) error {
	q.muEnq.Lock()
	defer q.muEnq.Unlock()

	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	if q.len == q.cap {
		return ErrQueueFull
	}

	q.items[q.tail] = item
	q.tail = (q.tail + 1) % q.cap
	q.len++
	q.cond.Signal()

	return nil
}

// Dequeue removes and returns an item from the queue. It returns an error if the queue is empty.
func (q *Queue[T]) Dequeue() (T, error) {
	q.muDeq.Lock()
	defer q.muDeq.Unlock()

	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	var zero T
	if q.len == 0 {
		return zero, ErrQueueEmpty
	}

	item := q.items[q.head]
	q.items[q.head] = zero // Clear the reference to allow garbage collection
	q.head = (q.head + 1) % q.cap
	q.len--
	q.cond.Signal()

	return item, nil
}

// BlockingEnqueue adds an item to the queue, blocking if the queue is full.
func (q *Queue[T]) BlockingEnqueue(item T) {
	q.muEnq.Lock()
	defer q.muEnq.Unlock()

	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for q.len == q.cap {
		q.cond.Wait()
	}

	q.items[q.tail] = item
	q.tail = (q.tail + 1) % q.cap
	q.len++
	q.cond.Signal()
}

// BlockingDequeue removes and returns an item from the queue, blocking if the queue is empty.
func (q *Queue[T]) BlockingDequeue() T {
	q.muDeq.Lock()
	defer q.muDeq.Unlock()

	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for q.len == 0 {
		q.cond.Wait()
	}

	var zero T
	item := q.items[q.head]
	q.items[q.head] = zero // Clear the reference to allow garbage collection
	q.head = (q.head + 1) % q.cap
	q.len--
	q.cond.Signal()

	return item
}

// Resize changes the capacity of the queue. It returns an error if the new capacity is smaller than the current number of items.
func (q *Queue[T]) Resize(newCap int) error {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

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
	q.cond.Broadcast()

	return nil
}
