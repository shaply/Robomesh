package data_structures

import (
	"sync"
	"sync/atomic"
)

type Node[T any] struct {
	value     T
	next      *Node[T]
	prev      *Node[T]
	rightLock sync.RWMutex // Lock for thread safety
	leftLock  sync.RWMutex // Lock for thread safety
	lock      sync.RWMutex // General lock for node operations
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

type SafeQueue[T any] struct {
	head *Node[T]
	tail *Node[T]
	len  atomic.Int64

	useWait   bool
	nextCh    chan bool
	readValCh chan bool
	notifyCh  chan bool
	done      chan struct{}
}

// This Set is a thread-safe data structure that allows multiple values of the same type to be stored.
// It is used for fast iterations, additions, and removals of values.
type SafeSet[T comparable] struct {
	mp      *SafeMap[T, *Node[T]]
	head    *Node[T]
	writeMu sync.Mutex // Lock for thread safety
}
