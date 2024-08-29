package fifo

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func assertDequeueList[T any](t *testing.T, q *Queue[T], expected []T, compare func(a, b T) bool) {
	t.Helper()
	for _, exp := range expected {
		item, err := q.TryDequeue()
		if err != nil || !compare(item, exp) {
			t.Fatalf("expected %v, got: %v, err: %v", exp, item, err)
		}
	}
}

func intCompare(a, b int) bool {
	return a == b
}

func TestQueue_NewWithNonPositiveCapacity(t *testing.T) {
	defer func() {
		if r := recover(); r != ErrCapacityNotPositive {
			t.Fatalf("expected panic with ErrCapacityNotPositive, got: %v", r)
		}
	}()

	New[int](0) // This should panic with ErrCapacityNotPositive
}

func TestQueue_LenCap(t *testing.T) {
	q := New[int](3)

	if q.Len() != 0 {
		t.Fatalf("expected len 0, got: %d", q.Len())
	}
	if q.Cap() != 3 {
		t.Fatalf("expected cap 3, got: %d", q.Cap())
	}

	q.TryEnqueue(1)
	q.TryEnqueue(2)

	if q.Len() != 2 {
		t.Fatalf("expected len 2, got: %d", q.Len())
	}
}

func TestQueue_TryEnqueueDequeue(t *testing.T) {
	q := New[int](3)

	// enqueue
	if err := q.TryEnqueue(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := q.TryEnqueue(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := q.TryEnqueue(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// enqueue after full
	if err := q.TryEnqueue(4); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got: %v", err)
	}

	// check data
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	// dequeue after empty
	_, err := q.TryDequeue()
	if err != ErrQueueEmpty {
		t.Fatalf("expected ErrQueueEmpty, got: %v", err)
	}
}

func TestQueue_ResizeUp(t *testing.T) {
	q := New[int](3)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)
	if err := q.TryEnqueue(4); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got: %v", err)
	}

	if err := q.Resize(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Cap() != 5 {
		t.Fatalf("expected cap 5, got: %d", q.Cap())
	}

	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	q.TryEnqueue(4)
	q.TryEnqueue(5)
	q.TryEnqueue(6)
	q.TryEnqueue(7)
	q.TryEnqueue(8)
	if err := q.TryEnqueue(9); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got: %v", err)
	}

	assertDequeueList(t, q, []int{4, 5, 6, 7, 8}, intCompare)
}

func TestQueue_ResizeToNonPositiveCapacity(t *testing.T) {
	q := New[int](3)

	err := q.Resize(0)
	if err != ErrCapacityNotPositive {
		t.Fatalf("expected ErrCapacityNotPositive, got: %v", err)
	}

	err = q.Resize(-1)
	if err != ErrCapacityNotPositive {
		t.Fatalf("expected ErrCapacityNotPositive, got: %v", err)
	}
}

func TestQueue_ResizeDown(t *testing.T) {
	q := New[int](5)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	if err := q.Resize(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Cap() != 3 {
		t.Fatalf("expected cap 3, got: %d", q.Cap())
	}

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	// Check new data works
	q.TryEnqueue(4)
	q.TryEnqueue(5)
	q.TryEnqueue(6)
	assertDequeueList(t, q, []int{4, 5, 6}, intCompare)

	// Ensure it's empty
	if _, err := q.TryDequeue(); err != ErrQueueEmpty {
		t.Fatalf("expected ErrQueueEmpty, got: %v", err)
	}
	if q.Len() != 0 {
		t.Fatalf("expected len 0, got: %d", q.Len())
	}
}

func TestQueue_ResizeEdgeCases(t *testing.T) {
	q := New[int](5)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	err := q.Resize(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Len() != 3 {
		t.Fatalf("expected len 3, got: %d", q.Len())
	}
	if q.Cap() != 2 {
		t.Fatalf("expected cap 2, got: %d", q.Cap())
	}

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	// Check new data works
	q.TryEnqueue(4)
	q.TryEnqueue(5)
	if err := q.TryEnqueue(6); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got: %v", err)
	}
	assertDequeueList(t, q, []int{4, 5}, intCompare)

	// Resize larger
	if err := q.Resize(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check more data works
	q.TryEnqueue(8)
	q.TryEnqueue(9)
	q.TryEnqueue(10)
	if err := q.TryEnqueue(11); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got: %v", err)
	}
	assertDequeueList(t, q, []int{8, 9, 10}, intCompare)

	// Ensure it's empty
	if _, err := q.TryDequeue(); err != ErrQueueEmpty {
		t.Fatalf("expected ErrQueueEmpty, got: %v", err)
	}
	if q.Len() != 0 {
		t.Fatalf("expected len 0, got: %d", q.Len())
	}
	if q.Cap() != 3 {
		t.Fatalf("expected cap 2, got: %d", q.Cap())
	}
}

func TestQueue_ResizeSameSize(t *testing.T) {
	q := New[int](3)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	if err := q.Resize(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Cap() != 3 {
		t.Fatalf("expected cap 3, got: %d", q.Cap())
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

func TestQueue_ConcurrentResize(t *testing.T) {
	q := New[int](10)

	for i := 0; i < 10; i++ {
		q.TryEnqueue(i)
	}

	var wg sync.WaitGroup

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

	for i := 0; i < 10; i++ {
		item, err := q.TryDequeue()
		if err != nil || item != i {
			t.Fatalf("expected %d, got: %v, err: %v", i, item, err)
		}
	}
}

func TestQueue_OperationsAfterClose(t *testing.T) {
	q := New[int](3)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	q.Close()

	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	if _, err := q.TryDequeue(); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}

	if err := q.TryEnqueue(4); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_ConcurrentTryEnqueueDequeue(t *testing.T) {
	q := New[int](100)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := q.TryEnqueue(n); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := q.TryDequeue()
			if err != nil && err != ErrQueueEmpty {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestQueue_ConcurrentBlockingEnqueueDequeue(t *testing.T) {
	q := New[int](100)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := q.Enqueue(n); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := q.Dequeue(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestQueue_Close(t *testing.T) {
	q := New[int](3)

	if err := q.Close(); err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}

	if err := q.Close(); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_TryEnqueueAfterClose(t *testing.T) {
	q := New[int](3)
	_ = q.Close()

	err := q.TryEnqueue(1)
	if err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_TryDequeueAfterClose(t *testing.T) {
	q := New[int](3)
	_ = q.Close()

	_, err := q.TryDequeue()
	if err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_BlockingEnqueueAfterClose(t *testing.T) {
	q := New[int](3)
	_ = q.Close()

	err := q.Enqueue(1)
	if err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_BlockingDequeueAfterClose(t *testing.T) {
	q := New[int](3)
	_ = q.Close()

	_, err := q.Dequeue()
	if err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_ConcurrentTryEnqueueAndClose(t *testing.T) {
	q := New[int](3)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := q.TryEnqueue(1)
		if err != nil && err != ErrQueueClosed {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		err := q.Close()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	wg.Wait()
}

func TestQueue_ConcurrentTryDequeueAndClose(t *testing.T) {
	q := New[int](3)
	q.TryEnqueue(1)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := q.TryDequeue()
		if err != nil && err != ErrQueueClosed {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		err := q.Close()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	wg.Wait()
}

func TestQueue_ConcurrentBlockingEnqueueAndClose(t *testing.T) {
	q := New[int](1)
	q.TryEnqueue(1)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := q.Enqueue(2)
		if err != nil && err != ErrQueueClosed {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		err := q.Close()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	wg.Wait()
}

func TestQueue_ConcurrentBlockingDequeueAndClose(t *testing.T) {
	q := New[int](3)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := q.Dequeue()
		if err != nil && err != ErrQueueClosed {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		err := q.Close()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	wg.Wait()
}

func TestQueue_OperationsAfterClosed(t *testing.T) {
	q := New[int](3)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	// Close the queue
	q.Close()

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	// Ensure it's empty
	if _, err := q.TryDequeue(); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}

	// Can't add more items
	if err := q.TryEnqueue(4); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}

	// Can't resize
	if err := q.Resize(5); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}

	// Can't close again
	if err := q.Close(); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_SmallConcurrentBlocking(t *testing.T) {
	num := 10
	q := New[int](5)
	var wg sync.WaitGroup
	var mu sync.Mutex
	inputs := []int{}
	outputs := []int{}

	// Concurrent Enqueue
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= num; i++ {
			if err := q.Enqueue(i); err != nil && err != ErrQueueClosed {
				t.Errorf("unexpected error: %v", err)
			} else {
				mu.Lock()
				inputs = append(inputs, i)
				mu.Unlock()
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Concurrent Dequeue
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= num; i++ {
			val, err := q.Dequeue()
			if err != nil && err != ErrQueueClosed {
				t.Errorf("unexpected error: %v", err)
			} else {
				mu.Lock()
				outputs = append(outputs, val)
				mu.Unlock()
			}
			time.Sleep(15 * time.Millisecond)
		}
	}()

	wg.Wait()

	if len(inputs) != len(outputs) {
		t.Fatalf("expected input length %d, got output length %d", len(inputs), len(outputs))
	}

	for i := 0; i < len(inputs); i++ {
		if inputs[i] != outputs[i] {
			t.Fatalf("expected output %d at index %d, got %d", inputs[i], i, outputs[i])
		}
	}
}

func TestQueue_MassConcurrentBlocking(t *testing.T) {
	q := New[int](10)
	const numGoroutines = 100
	const numItems = 1000

	var wg sync.WaitGroup

	// Concurrent Enqueue
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := start; j < start+numItems; j++ {
				if err := q.Enqueue(j); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}(i * numItems)
	}

	// Concurrent Dequeue
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numItems; j++ {
				_, err := q.Dequeue()
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}()
	}

	wg.Wait()
}

func ExampleQueue() {
	q := New[string](3)

	// Enqueue items
	q.TryEnqueue("A")
	q.TryEnqueue("B")
	q.TryEnqueue("C")

	fmt.Println("Length after enqueueing 3 items:", q.Len())
	fmt.Println("Capacity after enqueueing 3 items:", q.Cap())

	// Try to enqueue when full
	err := q.TryEnqueue("D")
	fmt.Println("TryEnqueue when full:", err)

	// Dequeue an item
	item, _ := q.TryDequeue()
	fmt.Println("Dequeued item:", item)

	// Enqueue another item
	q.TryEnqueue("D")

	// Resize the queue
	err = q.Resize(5)
	fmt.Println("Resize result:", err)
	fmt.Println("Capacity after resize:", q.Cap())

	// Enqueue more items
	q.TryEnqueue("E")
	q.TryEnqueue("F")

	// Dequeue all items
	for q.Len() > 0 {
		item, _ := q.TryDequeue()
		fmt.Println("Dequeued item:", item)
	}

	// Concurrent operations
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			q.Enqueue(fmt.Sprintf("G%d", i))
			time.Sleep(10 * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			item, _ := q.Dequeue()
			fmt.Println("Concurrent dequeued item:", item)
			time.Sleep(15 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Close the queue
	q.Close()

	// Try operations after closing
	err = q.TryEnqueue("H")
	fmt.Println("TryEnqueue after close:", err)

	_, err = q.TryDequeue()
	fmt.Println("TryDequeue after close:", err)

	err = q.Resize(10)
	fmt.Println("Resize after close:", err)

	// Output:
	// Length after enqueueing 3 items: 3
	// Capacity after enqueueing 3 items: 3
	// TryEnqueue when full: queue is full
	// Dequeued item: A
	// Resize result: <nil>
	// Capacity after resize: 5
	// Dequeued item: B
	// Dequeued item: C
	// Dequeued item: D
	// Dequeued item: E
	// Dequeued item: F
	// Concurrent dequeued item: G0
	// Concurrent dequeued item: G1
	// Concurrent dequeued item: G2
	// Concurrent dequeued item: G3
	// Concurrent dequeued item: G4
	// TryEnqueue after close: queue is closed
	// TryDequeue after close: queue is closed
	// Resize after close: queue is closed
}
