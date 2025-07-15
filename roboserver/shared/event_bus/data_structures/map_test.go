package data_structures

import (
	"sync"
	"testing"
)

func TestMapAdd(t *testing.T) {
	mp := &Map[string]{}
	node := &Node[string]{value: "test"}

	mp.Add(node)

	if mp.list == nil {
		t.Error("Expected list to not be nil after adding node")
	}

	if mp.list.node != node {
		t.Error("Expected list node to be the added node")
	}
}

func TestMapRemove(t *testing.T) {
	mp := &Map[string]{}
	node := &Node[string]{value: "test"}

	// Test remove from empty map
	result := mp.Remove("test")
	if result != nil {
		t.Error("Expected nil when removing from empty map")
	}

	// Add node and then remove
	mp.Add(node)
	result = mp.Remove("test")

	if result != node {
		t.Error("Expected to get back the same node")
	}
}

func TestMapConcurrency(t *testing.T) {
	mp := &Map[string]{}

	var wg sync.WaitGroup

	// Add nodes concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			node := &Node[string]{value: "test"}
			mp.Add(node)
		}(i)
	}

	wg.Wait()

	// Map should have nodes (exact count depends on implementation)
	if mp.list == nil {
		t.Error("Expected map to have nodes after concurrent adds")
	}
}
