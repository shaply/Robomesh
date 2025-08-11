package data_structures

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Basic functionality tests
func TestSafeMapBasicOperations(t *testing.T) {
	sm := NewSafeMap[string, int]()

	// Test Set and Get
	sm.Set("key1", 42)
	value, ok := sm.Get("key1")
	if !ok {
		t.Error("Expected to find key1")
	}
	if value != 42 {
		t.Errorf("Expected value 42, got %d", value)
	}

	// Test non-existent key
	_, ok = sm.Get("nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent key")
	}
}

func TestSafeMapGetOrDefault(t *testing.T) {
	sm := NewSafeMap[string, int]()

	// Test GetOrDefault with non-existent key
	value := sm.GetOrDefault("missing", 100)
	if value != 100 {
		t.Errorf("Expected default value 100, got %d", value)
	}

	// Check if the key was actually set
	storedValue, ok := sm.Get("missing")
	if !ok {
		t.Error("Expected key to be set by GetOrDefault")
	}
	if storedValue != 100 {
		t.Errorf("Expected stored value 100, got %d", storedValue)
	}

	// Test GetOrDefault with existing key
	sm.Set("existing", 50)
	value = sm.GetOrDefault("existing", 200)
	if value != 50 {
		t.Errorf("Expected existing value 50, got %d", value)
	}
}

func TestSafeMapDelete(t *testing.T) {
	sm := NewSafeMap[string, int]()

	// Set a value
	sm.Set("delete_me", 123)

	// Verify it exists
	_, ok := sm.Get("delete_me")
	if !ok {
		t.Error("Expected key to exist before deletion")
	}

	// Delete it
	sm.Delete("delete_me")

	// Verify it's gone
	_, ok = sm.Get("delete_me")
	if ok {
		t.Error("Expected key to be deleted")
	}

	// Delete non-existent key (should not panic)
	sm.Delete("never_existed")
}

func TestSafeMapNilMapInitialization(t *testing.T) {
	// Create SafeMap with nil internal map
	sm := &SafeMap[string, int]{}

	// Test Set initializes the map
	sm.Set("test", 42)

	value, ok := sm.Get("test")
	if !ok {
		t.Error("Expected to find key after Set on nil map")
	}
	if value != 42 {
		t.Errorf("Expected value 42, got %d", value)
	}
}

// Concurrency tests
func TestSafeMapConcurrentReadsWrites(t *testing.T) {
	sm := NewSafeMap[int, string]()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := id*numOperations + j
				sm.Set(key, fmt.Sprintf("value_%d", key))
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := id*numOperations + j
				sm.Get(key) // May or may not find the value
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	expectedCount := numGoroutines * numOperations
	actualCount := 0

	// Count all set values
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numOperations; j++ {
			key := i*numOperations + j
			if value, ok := sm.Get(key); ok {
				expectedValue := fmt.Sprintf("value_%d", key)
				if value != expectedValue {
					t.Errorf("Expected value %s, got %s", expectedValue, value)
				}
				actualCount++
			}
		}
	}

	if actualCount != expectedCount {
		t.Errorf("Expected %d values, found %d", expectedCount, actualCount)
	}
}

func TestSafeMapConcurrentGetOrDefault(t *testing.T) {
	sm := NewSafeMap[string, int]()

	var wg sync.WaitGroup
	var successCount int64
	numGoroutines := 50

	// Multiple goroutines try to set the same key with GetOrDefault
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// All goroutines try to set the same key
			value := sm.GetOrDefault("shared_key", id)

			// Only one should succeed in setting the initial value
			if value == id {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Only one goroutine should have successfully set the initial value
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful initial set, got %d", successCount)
	}

	// The key should exist
	value, ok := sm.Get("shared_key")
	if !ok {
		t.Error("Expected shared_key to exist")
	}

	// The value should be one of the goroutine IDs
	if value < 0 || value >= numGoroutines {
		t.Errorf("Expected value between 0 and %d, got %d", numGoroutines-1, value)
	}
}

func TestSafeMapConcurrentDeletes(t *testing.T) {
	sm := NewSafeMap[int, string]()

	// Pre-populate the map
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		sm.Set(i, fmt.Sprintf("value_%d", i))
	}

	var wg sync.WaitGroup

	// Concurrent deletes
	for i := 0; i < numKeys; i++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			sm.Delete(key)
		}(i)
	}

	wg.Wait()

	// All keys should be deleted
	for i := 0; i < numKeys; i++ {
		if _, ok := sm.Get(i); ok {
			t.Errorf("Expected key %d to be deleted", i)
		}
	}
}

// Edge case tests
func TestSafeMapEmptyOperations(t *testing.T) {
	sm := NewSafeMap[string, int]()

	// Get from empty map
	_, ok := sm.Get("empty")
	if ok {
		t.Error("Expected not to find key in empty map")
	}

	// GetOrDefault from empty map
	value := sm.GetOrDefault("empty", 42)
	if value != 42 {
		t.Errorf("Expected default value 42, got %d", value)
	}

	// Delete from empty map (should not panic)
	sm.Delete("nonexistent")
}

func TestSafeMapZeroValues(t *testing.T) {
	sm := NewSafeMap[string, int]()

	// Set zero value
	sm.Set("zero", 0)

	value, ok := sm.Get("zero")
	if !ok {
		t.Error("Expected to find zero value")
	}
	if value != 0 {
		t.Errorf("Expected zero value, got %d", value)
	}

	// GetOrDefault with zero value as default
	value = sm.GetOrDefault("another_zero", 0)
	if value != 0 {
		t.Errorf("Expected zero default value, got %d", value)
	}
}

func TestSafeMapStringKeys(t *testing.T) {
	sm := NewSafeMap[string, []int]()

	// Test with complex values
	slice := []int{1, 2, 3, 4, 5}
	sm.Set("slice_key", slice)

	retrievedSlice, ok := sm.Get("slice_key")
	if !ok {
		t.Error("Expected to find slice")
	}

	if len(retrievedSlice) != len(slice) {
		t.Errorf("Expected slice length %d, got %d", len(slice), len(retrievedSlice))
	}

	for i, v := range slice {
		if retrievedSlice[i] != v {
			t.Errorf("Expected slice[%d] = %d, got %d", i, v, retrievedSlice[i])
		}
	}
}

// Performance tests
func TestSafeMapPerformance(t *testing.T) {
	sm := NewSafeMap[int, string]()

	numOperations := 10000

	// Measure Set performance
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		sm.Set(i, fmt.Sprintf("value_%d", i))
	}
	setDuration := time.Since(start)

	// Measure Get performance
	start = time.Now()
	for i := 0; i < numOperations; i++ {
		sm.Get(i)
	}
	getDuration := time.Since(start)

	t.Logf("Set %d items in %v (avg: %v per item)",
		numOperations, setDuration, setDuration/time.Duration(numOperations))
	t.Logf("Get %d items in %v (avg: %v per item)",
		numOperations, getDuration, getDuration/time.Duration(numOperations))

	// Performance thresholds (adjust as needed)
	avgSetTime := setDuration / time.Duration(numOperations)
	avgGetTime := getDuration / time.Duration(numOperations)

	if avgSetTime > time.Microsecond*10 {
		t.Errorf("Set operation too slow: %v per item", avgSetTime)
	}
	if avgGetTime > time.Microsecond*5 {
		t.Errorf("Get operation too slow: %v per item", avgGetTime)
	}
}

// Race condition detection test
func TestSafeMapRaceDetection(t *testing.T) {
	// This test is designed to catch race conditions when run with -race flag
	sm := NewSafeMap[int, int]()

	var wg sync.WaitGroup
	numGoroutines := 20
	numOperations := 100

	// Mixed operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := (id * numOperations) + j

				// Mix of operations
				switch j % 4 {
				case 0:
					sm.Set(key, key*2)
				case 1:
					sm.Get(key)
				case 2:
					sm.GetOrDefault(key, key*3)
				case 3:
					sm.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()

	// Test passes if no race conditions are detected
}

// Benchmark tests
func BenchmarkSafeMapSet(b *testing.B) {
	sm := NewSafeMap[int, string]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.Set(i, fmt.Sprintf("value_%d", i))
	}
}

func BenchmarkSafeMapGet(b *testing.B) {
	sm := NewSafeMap[int, string]()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		sm.Set(i, fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.Get(i % 1000)
	}
}

func BenchmarkSafeMapGetOrDefault(b *testing.B) {
	sm := NewSafeMap[int, string]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.GetOrDefault(i, fmt.Sprintf("default_%d", i))
	}
}

func BenchmarkSafeMapConcurrentAccess(b *testing.B) {
	sm := NewSafeMap[int, string]()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		sm.Set(i, fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := i % 1000
			if i%2 == 0 {
				sm.Get(key)
			} else {
				sm.Set(key, fmt.Sprintf("new_value_%d", key))
			}
			i++
		}
	})
}
