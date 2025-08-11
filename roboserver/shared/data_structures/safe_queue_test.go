package data_structures

import (
	"testing"
	"time"
)

func TestSafeQueueBasicOperations(t *testing.T) {
	q := NewSafeQueue[int](false)

	// Test Enqueue and Dequeue
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	if value, ok := q.Dequeue(); !ok || value != 1 {
		t.Error("Expected to dequeue 1")
	}

	if value, ok := q.Dequeue(); !ok || value != 2 {
		t.Error("Expected to dequeue 2")
	}

	if value, ok := q.Dequeue(); !ok || value != 3 {
		t.Error("Expected to dequeue 3")
	}

	if _, ok := q.Dequeue(); ok {
		t.Error("Expected dequeue to fail on empty queue")
	}
}

func TestSafeQueueReadOperation(t *testing.T) {
	q := NewSafeQueue[int](true)

	// Test Enqueue
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	// Test Read
	if value, ok := q.Read(true); !ok || value != 1 {
		t.Error("Expected to read 1")
	}

	if value, ok := q.Read(true); !ok || value != 2 {
		t.Error("Expected to read 2")
	}

	if value, ok := q.Read(true); !ok || value != 3 {
		t.Error("Expected to read 3")
	}

	if q.Size() != 0 {
		t.Error("Expected queue size to be 0 after reading all elements")
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		q.Enqueue(4)
	}()

	// Simple timeout using select with time.After
	done := make(chan struct {
		value int
		ok    bool
	}, 1)

	go func() {
		value, ok := q.Read(true)
		done <- struct {
			value int
			ok    bool
		}{value, ok}
	}()

	select {
	case result := <-done:
		if !result.ok || result.value != 4 {
			t.Error("Expected to read 4")
		}
	case <-time.After(5000 * time.Millisecond):
		t.Error("Test timed out waiting for queue read")
	}
}

func TestSafeQueueConcurrentReadOperations(t *testing.T) {
	q := NewSafeQueue[int](true)
	defer q.Close()

	numReaders := 5
	numItems := 10
	results := make(chan int, numReaders*numItems)

	// Start multiple readers
	for i := 0; i < numReaders; i++ {
		go func() {
			for j := 0; j < numItems; j++ {
				if value, ok := q.Read(true); ok {
					results <- value
				}
			}
		}()
	}

	// Enqueue items with some delay
	go func() {
		for i := 0; i < numReaders*numItems; i++ {
			q.Enqueue(i)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Collect results with timeout
	collected := make(map[int]bool)
	timeout := time.After(5 * time.Second)

	for i := 0; i < numReaders*numItems; i++ {
		select {
		case value := <-results:
			collected[value] = true
		case <-timeout:
			t.Fatalf("Timeout: only collected %d out of %d items", len(collected), numReaders*numItems)
		}
	}

	// Verify all items were collected
	if len(collected) != numReaders*numItems {
		t.Errorf("Expected %d unique items, got %d", numReaders*numItems, len(collected))
	}
}

func TestSafeQueueNonBlockingRead(t *testing.T) {
	q := NewSafeQueue[string](true)
	defer q.Close()

	// Test non-blocking read on empty queue
	if value, ok := q.Read(false); ok {
		t.Errorf("Expected non-blocking read to fail on empty queue, got: %s", value)
	}

	// Add item and test non-blocking read
	q.Enqueue("test")
	time.Sleep(100 * time.Millisecond) // Ensure item is enqueued
	if value, ok := q.Read(false); !ok || value != "test" {
		t.Errorf("Expected non-blocking read to return 'test', got: %s, ok: %t", value, ok)
	}

	// Test non-blocking read on empty queue again
	if value, ok := q.Read(false); ok {
		t.Errorf("Expected non-blocking read to fail on empty queue after dequeue, got: %s", value)
	}
}

func TestSafeQueueWithEndChannel(t *testing.T) {
	q := NewSafeQueue[int](true)
	defer q.Close()

	endCh := make(chan struct{})

	// Test read with end channel that gets closed
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(endCh)
	}()

	if value, ok := q.Read(true, endCh); ok {
		t.Errorf("Expected read to be cancelled by end channel, got value: %d", value)
	}
}

func TestSafeQueueSize(t *testing.T) {
	q := NewSafeQueue[int](false)

	// Test empty queue size
	if size := q.Size(); size != 0 {
		t.Errorf("Expected empty queue size to be 0, got: %d", size)
	}

	// Test size after enqueuing
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	if size := q.Size(); size != 3 {
		t.Errorf("Expected queue size to be 3, got: %d", size)
	}

	// Test size after dequeuing
	q.Dequeue()
	if size := q.Size(); size != 2 {
		t.Errorf("Expected queue size to be 2 after dequeue, got: %d", size)
	}

	// Test size after dequeuing all
	q.Dequeue()
	q.Dequeue()
	if size := q.Size(); size != 0 {
		t.Errorf("Expected queue size to be 0 after dequeuing all, got: %d", size)
	}
}

func TestSafeQueueDifferentTypes(t *testing.T) {
	// Test with strings
	stringQueue := NewSafeQueue[string](false)
	stringQueue.Enqueue("hello")
	stringQueue.Enqueue("world")

	if value, ok := stringQueue.Dequeue(); !ok || value != "hello" {
		t.Errorf("Expected to dequeue 'hello', got: %s, ok: %t", value, ok)
	}

	// Test with structs
	type TestStruct struct {
		ID   int
		Name string
	}

	structQueue := NewSafeQueue[TestStruct](false)
	testItem := TestStruct{ID: 1, Name: "test"}
	structQueue.Enqueue(testItem)

	if value, ok := structQueue.Dequeue(); !ok || value.ID != 1 || value.Name != "test" {
		t.Errorf("Expected to dequeue TestStruct{1, 'test'}, got: %+v, ok: %t", value, ok)
	}
}

func TestSafeQueueConcurrentEnqueueDequeue(t *testing.T) {
	q := NewSafeQueue[int](false)
	numGoroutines := 10
	itemsPerGoroutine := 100

	var enqueued, dequeued int64

	// Start enqueuers
	for i := 0; i < numGoroutines; i++ {
		go func(start int) {
			for j := 0; j < itemsPerGoroutine; j++ {
				q.Enqueue(start*itemsPerGoroutine + j)
				enqueued++
			}
		}(i)
	}

	// Start dequeuers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < itemsPerGoroutine; j++ {
				for {
					if _, ok := q.Dequeue(); ok {
						dequeued++
						break
					}
					time.Sleep(1 * time.Millisecond) // Brief pause before retry
				}
			}
		}()
	}

	// Wait for completion
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout: enqueued %d, dequeued %d", enqueued, dequeued)
		case <-ticker.C:
			if dequeued == int64(numGoroutines*itemsPerGoroutine) {
				return // Test passed
			}
		}
	}
}

func TestSafeQueueClose(t *testing.T) {
	q := NewSafeQueue[int](true)

	// Test that queue can be closed without error
	if err := q.Close(); err != nil {
		t.Errorf("Expected Close() to return nil, got: %v", err)
	}

	// Test multiple closes don't panic
	if err := q.Close(); err != nil {
		t.Errorf("Expected second Close() to return nil, got: %v", err)
	}
}

func TestSafeQueueFIFOOrder(t *testing.T) {
	q := NewSafeQueue[int](false)

	// Enqueue in order
	for i := 0; i < 10; i++ {
		q.Enqueue(i)
	}

	// Dequeue and verify FIFO order
	for i := 0; i < 10; i++ {
		if value, ok := q.Dequeue(); !ok || value != i {
			t.Errorf("Expected to dequeue %d, got: %d, ok: %t", i, value, ok)
		}
	}
}

func TestSafeQueueEmptyDequeue(t *testing.T) {
	q := NewSafeQueue[int](false)

	// Test multiple dequeues on empty queue
	for i := 0; i < 5; i++ {
		if value, ok := q.Dequeue(); ok {
			t.Errorf("Expected dequeue to fail on empty queue, got: %d", value)
		}
	}
}

func TestSafeQueueWaitVsNonWaitBehavior(t *testing.T) {
	// Test non-wait queue
	nonWaitQ := NewSafeQueue[int](false)
	nonWaitQ.Enqueue(1)

	// Both Dequeue and Read should work the same for non-wait queue
	if value, ok := nonWaitQ.Read(true); !ok || value != 1 {
		t.Errorf("Expected Read to work like Dequeue for non-wait queue, got: %d, ok: %t", value, ok)
	}

	// Test wait queue
	waitQ := NewSafeQueue[int](true)
	defer waitQ.Close()

	waitQ.Enqueue(2)

	// For wait queue, Dequeue should call Read internally
	time.Sleep(100 * time.Millisecond) // Ensure item is enqueued
	if value, ok := waitQ.Dequeue(); !ok || value != 2 {
		t.Errorf("Expected Dequeue to work for wait queue, got: %d, ok: %t", value, ok)
	}
}
