package fifo

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// assertDequeueList is a helper function to assert that the dequeued items match the expected values.
func assertDequeueList[T any](t *testing.T, q *Queue[T], expected []T, compare func(a, b T) bool) {
	t.Helper() // Marks this function as a test helper function
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

func TestQueue_TryEnqueueDequeue(t *testing.T) {
	q := New[int](3)

	// Test TryEnqueue
	if err := q.TryEnqueue(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := q.TryEnqueue(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := q.TryEnqueue(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test TryEnqueue on full queue
	if err := q.TryEnqueue(4); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got: %v", err)
	}

	// Test TryDequeue
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	// Test TryDequeue on empty queue
	_, err := q.TryDequeue()
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

	q.TryEnqueue(1)
	q.TryEnqueue(2)

	if q.Len() != 2 {
		t.Fatalf("expected len 2, got: %d", q.Len())
	}
}

func TestQueue_ResizeUp(t *testing.T) {
	q := New[int](3)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	if err := q.Resize(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Cap() != 5 {
		t.Fatalf("expected cap 5, got: %d", q.Cap())
	}

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	// Check new data works
	q.TryEnqueue(4)
	q.TryEnqueue(5)
	q.TryEnqueue(6)
	q.TryEnqueue(7)
	q.TryEnqueue(8)

	assertDequeueList(t, q, []int{4, 5, 6, 7, 8}, intCompare)
}

func TestQueue_ResizeUp2(t *testing.T) {
	q := New[int](3)

	// Fill the queue
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	// TryDequeue two items
	assertDequeueList(t, q, []int{1, 2}, intCompare)

	// TryEnqueue two more items, causing wraparound
	q.TryEnqueue(4)
	q.TryEnqueue(5)

	// Resize the queue
	if err := q.Resize(5); err != nil {
		t.Fatalf("unexpected error on Resize: %v", err)
	}

	// Check if all items are preserved and in correct order
	assertDequeueList(t, q, []int{3, 4, 5}, intCompare)
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

func TestQueue_ResizeDown2(t *testing.T) {
	q := New[int](5)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	if err := q.Resize(4); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.Cap() != 4 {
		t.Fatalf("expected cap 4, got: %d", q.Cap())
	}

	// Ensure no data loss
	assertDequeueList(t, q, []int{1, 2, 3}, intCompare)

	// Check new data works
	q.TryEnqueue(4)
	q.TryEnqueue(5)
	q.TryEnqueue(6)
	q.TryEnqueue(7)
	assertDequeueList(t, q, []int{4, 5, 6}, intCompare)

	// Check more data works
	q.TryEnqueue(8)
	q.TryEnqueue(9)
	q.TryEnqueue(10)
	if err := q.TryEnqueue(11); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got: %v", err)
	}
	assertDequeueList(t, q, []int{7, 8, 9, 10}, intCompare)

	// Ensure it's empty
	if _, err := q.TryDequeue(); err != ErrQueueEmpty {
		t.Fatalf("expected ErrQueueEmpty, got: %v", err)
	}
	if q.Len() != 0 {
		t.Fatalf("expected len 0, got: %d", q.Len())
	}
}

func TestQueue_ResizeDown3(t *testing.T) {
	q := New[int](5)

	// Fill the queue
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)
	q.TryEnqueue(4)
	q.TryEnqueue(5)

	// TryDequeue two items
	assertDequeueList(t, q, []int{1, 2}, intCompare)

	// TryEnqueue two more items, causing wraparound
	q.TryEnqueue(6)

	// Resize the queue down
	if err := q.Resize(4); err != nil {
		t.Fatalf("unexpected error on Resize: %v", err)
	}

	// Check if all items are preserved and in correct order
	assertDequeueList(t, q, []int{3, 4, 5, 6}, intCompare)

	// Ensure it's empty
	if _, err := q.TryDequeue(); err != ErrQueueEmpty {
		t.Fatalf("expected ErrQueueEmpty, got: %v", err)
	}
	if q.Len() != 0 {
		t.Fatalf("expected len 0, got: %d", q.Len())
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

func TestQueue_ResizeTooSmall(t *testing.T) {
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

func TestQueue_ResizeEmptyQueue(t *testing.T) {
	q := New[int](5)

	if err := q.Resize(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Cap() != 3 {
		t.Fatalf("expected cap 3, got: %d", q.Cap())
	}
}

func TestQueue_ConcurrentTryEnqueueDequeue(t *testing.T) {
	q := New[int](100)

	var wg sync.WaitGroup

	// Concurrent TryEnqueue
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := q.TryEnqueue(n); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	// Concurrent TryDequeue
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

	// Concurrent Enqueue
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if err := q.BlockingEnqueue(n); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	// Concurrent Dequeue
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := q.BlockingDequeue(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestQueue_ConcurrentResize(t *testing.T) {
	q := New[int](10)

	for i := 0; i < 10; i++ {
		q.TryEnqueue(i)
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
		item, err := q.TryDequeue()
		if err != nil || item != i {
			t.Fatalf("expected %d, got: %v, err: %v", i, item, err)
		}
	}
}

func TestQueue_Blocking(t *testing.T) {
	q := New[int](1)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		q.TryEnqueue(1)
		t.Logf("just enqueued 1")

		q.BlockingEnqueue(2)
		t.Logf("blocking enqueued 2")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(100 * time.Millisecond)
		item, err := q.TryDequeue()
		t.Logf("just dequeued %d -- %v", item, err)

		item, _ = q.BlockingDequeue()
		t.Logf("blocking dequeued %d", item)
	}()

	wg.Wait()
}

func TestQueue_TryEnqueueLen(t *testing.T) {
	q := New[int](5)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 12; i++ {
			t.Logf("[%d] len: %d", i, q.Len())
			time.Sleep(3 * time.Millisecond)
		}
	}()

	for i := 1; i <= 6; i++ {
		time.Sleep(5 * time.Millisecond)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q.TryEnqueue(i)
		}(i)
	}

	wg.Wait()
}

func TestQueue_TryDequeueLen(t *testing.T) {
	q := New[int](5)
	for i := 1; i <= 6; i++ {
		q.TryEnqueue(i)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 12; i++ {
			t.Logf("[%d] len: %d", i, q.Len())
			time.Sleep(3 * time.Millisecond)
		}
	}()

	for i := 1; i <= 6; i++ {
		time.Sleep(5 * time.Millisecond)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q.TryDequeue()
		}(i)
	}
	wg.Wait()
}

func TestQueue_BlockingDequeueEnqueue(t *testing.T) {
	q := New[int](1)
	times := 20

	var wg sync.WaitGroup
	for i := 0; i < times; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			//t.Logf("blocking dequeued %d", idx)
			//fmt.Printf("blocking dequeued %d\n", idx)
			q.BlockingDequeue()
		}(i)
	}

	time.Sleep(10 * time.Millisecond)
	for i := 0; i < times; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			//t.Logf("blocking enqueued %d", idx)
			//fmt.Printf("blocking enqueued %d\n", idx)
			q.BlockingEnqueue(idx)
		}(i)
	}

	wg.Wait()
}

func TestQueue_BlockingEnqueueDequeue(t *testing.T) {
	q := New[int](1)
	times := 20

	var wg sync.WaitGroup
	for i := 0; i < times; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			//t.Logf("blocking enqueued %d", idx)
			//fmt.Printf("blocking enqueued %d\n", idx)
			q.BlockingEnqueue(idx)
		}(i)
	}

	time.Sleep(10 * time.Millisecond)
	for i := 0; i < times; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			//t.Logf("blocking dequeued %d", idx)
			//fmt.Printf("blocking dequeued %d\n", idx)
			q.BlockingDequeue()
		}(i)
	}

	wg.Wait()
}

func TestQueue_BlockingEnqueueResize(t *testing.T) {
	q := New[int](1)
	times := 5

	var wg sync.WaitGroup
	for i := 0; i < times; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			q.BlockingEnqueue(idx)
		}(i)
	}

	// Resize the queue
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < times; i++ {
		q.Resize(2 + i)
	}

	// Wait for all enqueue operations to complete
	wg.Wait()

	// Verify the queue contents
	if q.Len() != times {
		t.Errorf("Expected queue length %d, got %d", times, q.Len())
	}

	seen := make(map[int]bool)
	for i := 0; i < times; i++ {
		item, err := q.TryDequeue()
		if err != nil {
			t.Errorf("Unexpected error during TryDequeue: %v", err)
		}
		if seen[item] {
			t.Errorf("Duplicate item found: %d", item)
		}
		seen[item] = true
	}

	if q.Len() != 0 {
		t.Errorf("Expected empty queue, got length %d", q.Len())
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

	err := q.BlockingEnqueue(1)
	if err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_BlockingDequeueAfterClose(t *testing.T) {
	q := New[int](3)
	_ = q.Close()

	_, err := q.BlockingDequeue()
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
		err := q.BlockingEnqueue(2)
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
		_, err := q.BlockingDequeue()
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

func TestQueue_Close(t *testing.T) {
	q := New[int](3)

	// Test closing an open queue
	if err := q.Close(); err != nil {
		t.Fatalf("unexpected error on Close: %v", err)
	}

	// Test closing an already closed queue
	if err := q.Close(); err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
}

func TestQueue_CloseUnblocksBlockingEnqueue(t *testing.T) {
	q := New[int](1)
	q.TryEnqueue(1)

	done := make(chan struct{})
	go func() {
		err := q.BlockingEnqueue(2)
		if err != ErrQueueClosed {
			t.Errorf("expected ErrQueueClosed, got: %v", err)
		}
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	q.Close()

	select {
	case <-done:
		// Test passed
	case <-time.After(1 * time.Second):
		t.Fatal("BlockingEnqueue was not unblocked by Close")
	}
}

func TestQueue_CloseUnblocksBlockingDequeue(t *testing.T) {
	q := New[int](1)

	done := make(chan struct{})
	go func() {
		_, err := q.BlockingDequeue()
		if err != ErrQueueClosed {
			t.Errorf("expected ErrQueueClosed, got: %v", err)
		}
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	q.Close()

	select {
	case <-done:
		// Test passed
	case <-time.After(1 * time.Second):
		t.Fatal("BlockingDequeue was not unblocked by Close")
	}
}

func TestQueue_EnqueueDequeueWithWraparound(t *testing.T) {
	q := New[int](3)

	// Fill the queue
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	// TryDequeue two items
	assertDequeueList(t, q, []int{1, 2}, intCompare)

	// TryEnqueue two more items, causing wraparound
	q.TryEnqueue(4)
	q.TryEnqueue(5)

	// TryDequeue all items and check order
	assertDequeueList(t, q, []int{3, 4, 5}, intCompare)
}

func TestQueue_MassConcurrentBlocking(t *testing.T) {
	q := New[int](10)
	const numGoroutines = 100
	const numItems = 1000

	var wg sync.WaitGroup

	// Concurrent BlockingEnqueue
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := start; j < start+numItems; j++ {
				if err := q.BlockingEnqueue(j); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}(i * numItems)
	}

	// Concurrent BlockingDequeue
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numItems; j++ {
				_, err := q.BlockingDequeue()
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}()
	}

	wg.Wait()
}

func TestQueue_SmallConcurrentBlocking(t *testing.T) {
	num := 10
	q := New[int](5)
	var wg sync.WaitGroup
	var mu sync.Mutex
	inputs := []int{}
	outputs := []int{}

	// Concurrent BlockingEnqueue
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= num; i++ {
			if err := q.BlockingEnqueue(i); err != nil && err != ErrQueueClosed {
				t.Errorf("unexpected error: %v", err)
			} else {
				mu.Lock()
				inputs = append(inputs, i)
				mu.Unlock()
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Concurrent BlockingDequeue
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= num; i++ {
			val, err := q.BlockingDequeue()
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

func TestQueue_OneCapacity(t *testing.T) {
	q := New[int](1)

	var wg sync.WaitGroup
	wg.Add(2)

	// TryEnqueue in a separate goroutine
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		err := q.BlockingEnqueue(1)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	// TryDequeue in the main goroutine
	go func() {
		defer wg.Done()
		val, err := q.BlockingDequeue()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != 1 {
			t.Fatalf("expected 1, got %d", val)
		}
	}()

	wg.Wait()
}

func TestQueue_NewWithNonPositiveCapacity(t *testing.T) {
	defer func() {
		if r := recover(); r != ErrCapacityNotPositive {
			t.Fatalf("expected panic with ErrCapacityNotPositive, got: %v", r)
		}
	}()

	New[int](0) // This should panic with ErrCapacityNotPositive
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

func TestQueue_ResizeWhenClosed(t *testing.T) {
	q := New[int](3)
	q.TryEnqueue(1)
	q.TryEnqueue(2)
	q.TryEnqueue(3)

	// Close the queue
	q.Close()

	err := q.Resize(5)
	if err != ErrQueueClosed {
		t.Fatalf("expected ErrQueueClosed, got: %v", err)
	}
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

func ExampleQueue() {
	q := New[string](3)

	q.TryEnqueue("A")
	q.TryEnqueue("B")
	q.TryEnqueue("C")

	fmt.Println("Length:", q.Len())
	fmt.Println("Capacity:", q.Cap())

	fmt.Println("Exceeded:", q.TryEnqueue("X"))

	item, _ := q.TryDequeue()
	fmt.Println("Dequeued:", item)

	q.TryEnqueue("D")
	fmt.Println("Resized:", q.Resize(4))
	fmt.Println("Length:", q.Len())
	fmt.Println("Capacity:", q.Cap())
	q.TryEnqueue("E")

	for q.Len() > 0 {
		item, _ := q.TryDequeue()
		fmt.Println("Dequeued:", item)
	}

	// Output:
	// Length: 3
	// Capacity: 3
	// Exceeded: queue is full
	// Dequeued: A
	// Resized: <nil>
	// Length: 3
	// Capacity: 4
	// Dequeued: B
	// Dequeued: C
	// Dequeued: D
	// Dequeued: E
}
