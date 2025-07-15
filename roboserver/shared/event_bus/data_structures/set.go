package data_structures

// NewSet creates a new Set instance
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		mp:   NewSafeMap[T, *Node[T]](),
		head: &Node[T]{},
	}
}

// Add inserts a new value into the set and updates the map
func (s *Set[T]) Add(value T) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, exists := s.mp.Get(value); exists {
		return // Value already exists in the set
	}
	n := s.head.AddRight(value)
	s.mp.Set(value, n)
}

// Remove deletes a value from the set and cleans up the map if necessary
func (s *Set[T]) Remove(value T) {
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

// Iterate returns a channel that yields all values in the set
// Usage: for value := range set.Iterate() { ... }
func (s *Set[T]) Iterate() <-chan T {
	ch := make(chan T)
	go func(ch chan T) {
		defer close(ch)
		for node := s.head.next; node != nil; node = node.next {
			node.lock.RLock()
			defer node.lock.RUnlock()
			ch <- node.value
		}
	}(ch)
	return ch
}
