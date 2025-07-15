package data_structures

import "sync"

type Node[T any] struct {
	value T
	next  *Node[T]
	prev  *Node[T]
	lock  sync.RWMutex // Lock for thread safety
}
type MapNode[T any] struct {
	node *Node[T]
	next *MapNode[T]
}
type Map[T comparable] struct {
	list *MapNode[T]
	lock sync.Mutex // Lock for thread safety
}
type SafeMap[K comparable, V any] struct {
	m  map[K]V
	mu sync.RWMutex
}

// This Set is a thread-safe data structure that allows multiple values of the same type to be stored.
// It is used for fast iterations, additions, and removals of values.
type Set[T comparable] struct {
	mp      *SafeMap[T, *Node[T]]
	head    *Node[T]
	writeMu sync.Mutex // Lock for thread safety
}
