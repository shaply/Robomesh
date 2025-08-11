package data_structures

func (mp *Map[T]) Add(node *Node[T]) {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	if mp.list == nil {
		mp.list = &MapNode[T]{node: node}
	} else {
		mp.list = &MapNode[T]{node: node, next: mp.list}
	}
}

func (mp *Map[T]) Remove(value T) *Node[T] {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	if mp.list == nil {
		return nil
	}

	mpNode := mp.list
	mp.list = mpNode.next
	return mpNode.node
}
