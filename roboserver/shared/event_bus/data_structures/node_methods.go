package data_structures

func (n *Node[T]) AddRight(value T) *Node[T] {
	if n.next != nil {
		n.next.lock.Lock()
		defer n.next.lock.Unlock()
	}
	n.lock.Lock()
	defer n.lock.Unlock()

	newNode := &Node[T]{value: value}
	newNode.prev = n
	newNode.next = n.next
	if n.next != nil {
		n.next.prev = newNode
	}
	n.next = newNode
	return newNode
}

func (n *Node[T]) RemoveSelf() {
	if n.next != nil {
		n.next.lock.Lock()
		defer n.next.lock.Unlock()
	}
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.prev != nil {
		n.prev.lock.Lock()
		defer n.prev.lock.Unlock()
	}

	if n.prev != nil && n.prev.next == n {
		n.prev.next = n.next
	}
	if n.next != nil && n.next.prev == n {
		n.next.prev = n.prev
	}
}
