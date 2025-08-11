package data_structures

import "roboserver/shared/utils"

func (n *Node[T]) AddRight(value T) *Node[T] {
	ch, done := n.manageRightLocks()
	<-ch
	defer utils.SafeCloseChannel(done)

	newNode := &Node[T]{value: value}
	newNode.prev = n
	newNode.next = n.next
	if n.next != nil {
		n.next.prev = newNode
	}
	n.next = newNode
	return newNode
}

func (n *Node[T]) AddLeft(value T) *Node[T] {
	ch, done := n.manageLeftLocks()
	<-ch
	defer utils.SafeCloseChannel(done)

	newNode := &Node[T]{value: value}
	newNode.next = n
	newNode.prev = n.prev
	if n.prev != nil {
		n.prev.next = newNode
	}
	n.prev = newNode
	return newNode
}

func (n *Node[T]) RemoveSelf() T {
	rch, rdone := n.manageRightLocks()
	lch, ldone := n.manageLeftLocks()
	<-rch
	<-lch
	defer utils.SafeCloseChannel(rdone)
	defer utils.SafeCloseChannel(ldone)
	return n._RemoveSelf()
}

func (n *Node[T]) _RemoveSelf() T {
	if n.prev != nil {
		n.prev.next = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	}
	return n.value
}

func (n *Node[T]) GetValue() T {
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.value
}

func (n *Node[T]) SetValue(value T) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.value = value
}

func (n *Node[T]) GetRight() *Node[T] {
	ch, done := n.manageRightLocks()
	<-ch
	defer utils.SafeCloseChannel(done)
	return n.next
}

func (n *Node[T]) GetLeft() *Node[T] {
	ch, done := n.manageLeftLocks()
	<-ch
	defer utils.SafeCloseChannel(done)
	return n.prev
}

// Locks the left lock of node and right lock of left node
// Pushes true to the channel when successfully locked
// First channel is for successful locking, second channel (close it) is for unlocking
func (n *Node[T]) manageLeftLocks() (chan bool, chan bool) {
	ch := make(chan bool)
	done := make(chan bool)
	go func() {
		defer utils.SafeCloseChannel(ch)
		defer utils.SafeCloseChannel(done)
		if n.prev != nil {
			l := n.prev
			l.rightLock.Lock()
			for l != nil && l != n.prev {
				l.rightLock.Unlock()
				l = n.prev
				l.rightLock.Lock()
			}
			if l != nil {
				defer l.rightLock.Unlock()
			}
		}

		n.leftLock.Lock()
		defer n.leftLock.Unlock()

		ch <- true

		<-done
	}()
	return ch, done
}

// First channel is for successful locking, second channel (close it) is for unlocking
func (n *Node[T]) manageRightLocks() (chan bool, chan bool) {
	ch := make(chan bool)
	done := make(chan bool)
	go func() {
		defer utils.SafeCloseChannel(ch)
		defer utils.SafeCloseChannel(done)
		n.rightLock.Lock()
		defer n.rightLock.Unlock()

		if n.next != nil {
			r := n.next
			r.leftLock.Lock()
			defer r.leftLock.Unlock()
		}

		ch <- true

		<-done
	}()
	return ch, done
}
