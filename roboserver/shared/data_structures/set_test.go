package data_structures

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

// Basic functionality tests
func TestSetAdd(t *testing.T) {
	set := NewSafeSet[string]()
	value := "test"

	set.Add(value)
	set.Add("another")
	set.Add("test") // Adding the same value again
	set.Add("bruh")

	ch := set.Iterate()
	count := 0
	for v := range ch {
		count++
		if v == value {
			continue
		}
		if v != "another" && v != "bruh" && v != "test" {
			t.Errorf("Unexpected value in set: %s", v)
		}
	}
	if count != 3 {
		t.Errorf("Expected 3 values in set, got %d", count)
	}
}

func TestSetRemove(t *testing.T) {
	set := NewSafeSet[string]()
	value := "test"
	set.Add(value)
	set.Add("another")

	set.Remove(value)

	ch := set.Iterate()
	count := 0
	for v := range ch {
		count++
		if v == value {
			t.Errorf("Value %s should have been removed", value)
		}
	}
	if count != 1 {
		t.Errorf("Expected 1 value in set after removal, got %d", count)
	}
}

// Edge case tests
func TestSetEmpty(t *testing.T) {
	set := NewSafeSet[string]()

	// Test iteration on empty set
	ch := set.Iterate()
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("Expected 0 values in empty set, got %d", count)
	}

	// Test removal from empty set (should not panic)
	set.Remove("nonexistent")

	// Verify still empty
	ch = set.Iterate()
	for range ch {
		t.Error("Expected empty set after removing from empty set")
	}
}

func TestSetRemoveNonexistent(t *testing.T) {
	set := NewSafeSet[string]()
	set.Add("exists")

	// Remove something that doesn't exist
	set.Remove("nonexistent")

	// Verify original item still exists
	ch := set.Iterate()
	count := 0
	found := false
	for v := range ch {
		count++
		if v == "exists" {
			found = true
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 item after removing nonexistent, got %d", count)
	}
	if !found {
		t.Error("Original item should still exist after removing nonexistent item")
	}
}

func TestSetDuplicateAdditions(t *testing.T) {
	set := NewSafeSet[int]()

	// Add same value multiple times
	for i := 0; i < 100; i++ {
		set.Add(42)
	}

	// Should only have one instance
	ch := set.Iterate()
	count := 0
	for v := range ch {
		count++
		if v != 42 {
			t.Errorf("Expected only value 42, got %d", v)
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 unique value after 100 duplicate additions, got %d", count)
	}
}

func TestSetWithDifferentTypes(t *testing.T) {
	// Test with integers
	intSet := NewSafeSet[int]()
	intSet.Add(1)
	intSet.Add(2)
	intSet.Add(3)

	intCount := 0
	for range intSet.Iterate() {
		intCount++
	}
	if intCount != 3 {
		t.Errorf("Expected 3 integers, got %d", intCount)
	}

	// Test with floats
	floatSet := NewSafeSet[float64]()
	floatSet.Add(1.5)
	floatSet.Add(2.7)
	floatSet.Add(1.5) // duplicate

	floatCount := 0
	for range floatSet.Iterate() {
		floatCount++
	}
	if floatCount != 2 {
		t.Errorf("Expected 2 unique floats, got %d", floatCount)
	}
}

// Concurrency tests
func TestSetConcurrentAdds(t *testing.T) {
	set := NewSafeSet[int]()
	var wg sync.WaitGroup
	numGoroutines := 100
	itemsPerGoroutine := 10

	// Add items concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				// Add unique values per goroutine
				set.Add(goroutineID*itemsPerGoroutine + j)
			}
		}(i)
	}

	wg.Wait()

	// Count total items
	ch := set.Iterate()
	count := 0
	seen := make(map[int]bool)
	for v := range ch {
		if seen[v] {
			t.Errorf("Duplicate value found: %d", v)
		}
		seen[v] = true
		count++
	}

	expected := numGoroutines * itemsPerGoroutine
	if count != expected {
		t.Errorf("Expected %d unique items, got %d", expected, count)
	}
}

func TestSetConcurrentRemoves(t *testing.T) {
	set := NewSafeSet[int]()

	// Pre-populate set
	for i := 0; i < 100; i++ {
		set.Add(i)
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Remove items concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				set.Remove(goroutineID*10 + j)
			}
		}(i)
	}

	wg.Wait()

	// Should be empty
	ch := set.Iterate()
	count := 0
	for range ch {
		count++
	}

	if count != 0 {
		t.Errorf("Expected empty set after concurrent removes, got %d items", count)
	}
}

func TestSetConcurrentAddRemove(t *testing.T) {
	set := NewSafeSet[int]()
	var wg sync.WaitGroup

	// Mixed concurrent operations
	for i := 0; i < 50; i++ {
		wg.Add(2)

		// Adder goroutine
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				set.Add(id*10 + j)
			}
		}(i)

		// Remover goroutine
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				set.Remove(id*10 + j)
			}
		}(i)
	}

	wg.Wait()

	// Should have some items (exact count depends on timing)
	ch := set.Iterate()
	count := 0
	for range ch {
		count++
	}

	// Should have items but less than total added
	if count < 0 || count > 500 {
		t.Errorf("Expected reasonable count after mixed operations, got %d", count)
	}
}

// Performance tests
func TestSetLargeDataset(t *testing.T) {
	set := NewSafeSet[int]()
	size := 10000

	start := time.Now()

	// Add many items
	for i := 0; i < size; i++ {
		set.Add(i)
	}

	addDuration := time.Since(start)

	// Iterate through all items
	start = time.Now()
	ch := set.Iterate()
	count := 0
	for range ch {
		count++
	}
	iterateDuration := time.Since(start)

	if count != size {
		t.Errorf("Expected %d items, got %d", size, count)
	}

	t.Logf("Added %d items in %v", size, addDuration)
	t.Logf("Iterated %d items in %v", size, iterateDuration)

	// Performance check (adjust thresholds as needed)
	if addDuration > time.Second {
		t.Errorf("Adding %d items took too long: %v", size, addDuration)
	}
}

func TestSetRandomOperations(t *testing.T) {
	set := NewSafeSet[int]()
	rand.Seed(time.Now().UnixNano())

	operations := 1000
	valueRange := 100

	for i := 0; i < operations; i++ {
		value := rand.Intn(valueRange)

		if rand.Float32() < 0.7 { // 70% chance to add
			set.Add(value)
		} else { // 30% chance to remove
			set.Remove(value)
		}
	}

	// Verify set consistency
	ch := set.Iterate()
	seen := make(map[int]bool)
	count := 0

	for v := range ch {
		if seen[v] {
			t.Errorf("Duplicate value in set: %d", v)
		}
		seen[v] = true
		count++

		if v < 0 || v >= valueRange {
			t.Errorf("Value out of expected range: %d", v)
		}
	}

	t.Logf("After %d random operations, set has %d unique items", operations, count)
}

// Benchmark tests
func BenchmarkSetAdd(b *testing.B) {
	set := NewSafeSet[int]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Add(i)
	}
}

func BenchmarkSetAddDuplicates(b *testing.B) {
	set := NewSafeSet[int]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Add(i % 100) // Only 100 unique values
	}
}

func BenchmarkSetRemove(b *testing.B) {
	set := NewSafeSet[int]()

	// Pre-populate
	for i := 0; i < b.N; i++ {
		set.Add(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Remove(i)
	}
}

func BenchmarkSetIterate(b *testing.B) {
	set := NewSafeSet[int]()

	// Pre-populate with 1000 items
	for i := 0; i < 1000; i++ {
		set.Add(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := set.Iterate()
		for range ch {
			// Just iterate, don't do anything
		}
	}
}

// Race condition detection test
func TestSetRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	set := NewSafeSet[int]()
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			set.Add(i)
			if i%2 == 0 {
				set.Remove(i / 2)
			}
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			ch := set.Iterate()
			for range ch {
				// Just read
			}
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Final consistency check
	ch := set.Iterate()
	seen := make(map[int]bool)
	for v := range ch {
		if seen[v] {
			t.Errorf("Race condition detected: duplicate value %d", v)
		}
		seen[v] = true
	}
}
