package fifo

import (
	"fmt"
	"sync"
	"testing"
)

// assertDequeueList is a helper function to assert that the dequeued items match the expected values.
func assertDequeueList[T any](t *testing.T, q *Queue[T], expected []T, compare func(a, b T) bool) {
	t.Helper() // Marks this function as a test helper function
	for _, exp := range expected {
		item, err := q.Dequeue()
		if err != nil || !compare(item, exp) {
			t.Fatalf("expected %v, got: %v, err: %v", exp, item, err)
		}
	}
}

func intCompare(a, b int) bool {
	return a == b
}

func TestQueue_EnqueueDequeue(t *testing.T) {
	q := New[int](3)

	// Test Enqueue
	if err := q.Enqueue(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := q.Enqueue(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := q.Enqueue(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Enqueue on full queue
	if err := q.Enqueue(4); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got: %v", err)
	}

	// Test Dequeue
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	// Test Dequeue on empty queue
	_, err := q.Dequeue()
	if err != ErrQueueEmpty {
		t.Fatalf("expected ErrQueueEmpty, got: %v", err)
	}
}

func TestQueue_LenCap(t *testing.T) {
	q := New[int](3)

	if q.Len() != 0 {
		t.Fatalf("expected len 0, got: %d", q.Len())
	}
	if q.Cap() != 3 {
		t.Fatalf("expected cap 3, got: %d", q.Cap())
	}

	q.Enqueue(1)
	q.Enqueue(2)

	if q.Len() != 2 {
		t.Fatalf("expected len 2, got: %d", q.Len())
	}
}

func TestQueue_ResizeUp(t *testing.T) {
	q := New[int](3)
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	if err := q.Resize(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Cap() != 5 {
		t.Fatalf("expected cap 5, got: %d", q.Cap())
	}

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)
}

func TestQueue_ResizeDown(t *testing.T) {
	q := New[int](5)
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	if err := q.Resize(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Cap() != 3 {
		t.Fatalf("expected cap 3, got: %d", q.Cap())
	}

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)
}

func TestQueue_ResizeSameSize(t *testing.T) {
	q := New[int](3)
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	if err := q.Resize(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Cap() != 3 {
		t.Fatalf("expected cap 3, got: %d", q.Cap())
	}

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)
}

func TestQueue_ResizeTooSmall(t *testing.T) {
	q := New[int](5)
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	err := q.Resize(2)
	if err != ErrNewCapacityTooSmall {
		t.Fatalf("expected ErrNewCapacityTooSmall, got: %v", err)
	}

	if q.Cap() != 5 {
		t.Fatalf("expected cap 5, got: %d", q.Cap())
	}

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)
}

func TestQueue_ResizeEmptyQueue(t *testing.T) {
	q := New[int](5)

	if err := q.Resize(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Cap() != 3 {
		t.Fatalf("expected cap 3, got: %d", q.Cap())
	}
}

func TestQueue_ConcurrentEnqueueDequeue(t *testing.T) {
	q := New[int](100)

	var wg sync.WaitGroup

	// Concurrent Enqueue
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := q.Enqueue(n); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	// Concurrent Dequeue
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := q.Dequeue()
			if err != nil && err != ErrQueueEmpty {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestQueue_ConcurrentResize(t *testing.T) {
	q := New[int](10)

	for i := 0; i < 10; i++ {
		q.Enqueue(i)
	}

	var wg sync.WaitGroup

	// Concurrent Resize
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := q.Resize(20); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if err := q.Resize(10); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	// Ensure no data loss
	for i := 0; i < 10; i++ {
		item, err := q.Dequeue()
		if err != nil || item != i {
			t.Fatalf("expected %d, got: %v, err: %v", i, item, err)
		}
	}
}

func ExampleQueue() {
	q := New[string](3)

	q.Enqueue("A")
	q.Enqueue("B")
	q.Enqueue("C")

	fmt.Println("Length:", q.Len())
	fmt.Println("Capacity:", q.Cap())

	fmt.Println("Exceeded:", q.Enqueue("X"))

	item, _ := q.Dequeue()
	fmt.Println("Dequeued:", item)

	q.Enqueue("D")

	for q.Len() > 0 {
		item, _ := q.Dequeue()
		fmt.Println("Dequeued:", item)
	}

	// Output:
	// Length: 3
	// Capacity: 3
	// Exceeded: queue is full
	// Dequeued: A
	// Dequeued: B
	// Dequeued: C
	// Dequeued: D
}
