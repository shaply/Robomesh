package data_structures

// NewSafeSet creates a new SafeSet instance
func NewSafeSet[T comparable]() *SafeSet[T] {
	return &SafeSet[T]{
		mp:   NewSafeMap[T, *Node[T]](),
		head: &Node[T]{},
	}
}

// Add inserts a new value into the set and updates the map
func (s *SafeSet[T]) Add(value T) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, exists := s.mp.Get(value); exists {
		return // Value already exists in the set
	}
	n := s.head.AddRight(value)
	s.mp.Set(value, n)
}

// Remove deletes a value from the set
func (s *SafeSet[T]) Remove(value T) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n, ok := s.mp.Get(value)
	if !ok {
		return
	}
	if n != nil {
		n.RemoveSelf()
	}
	s.mp.Delete(value)
}

// Iterate returns a channel that yields all values in the set.
// Implemented as a snapshot so early break by the caller does not leak a
// goroutine blocked on a channel send.
// Usage: for value := range set.Iterate() { ... }
func (s *SafeSet[T]) Iterate() <-chan T {
	snap := s.Snapshot()
	ch := make(chan T, len(snap))
	for _, v := range snap {
		ch <- v
	}
	close(ch)
	return ch
}

// Snapshot returns a point-in-time copy of all values in the set.
// Safe to traverse without holding any lock.
func (s *SafeSet[T]) Snapshot() []T {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	out := make([]T, 0)
	for node := s.head.next; node != nil; node = node.next {
		out = append(out, node.value)
	}
	return out
}

func (s *SafeSet[T]) IsEmpty() bool {
	return s.mp.IsEmpty()
}

func (s *SafeSet[T]) Contains(value T) bool {
	_, exists := s.mp.Get(value)
	return exists
}
