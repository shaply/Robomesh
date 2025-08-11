package data_structures

import (
	"roboserver/shared/utils"
)

// Maybe add switching between using the go routine and not using it

// If using wait, the queue must be closed when done to avoid goroutine leaks
func NewSafeQueue[T any](useWait bool) *SafeQueue[T] {
	q := &SafeQueue[T]{
		head: &Node[T]{}, // Dummy head node
		tail: &Node[T]{}, // Dummy tail node
	}
	q.head.next = q.tail
	q.tail.prev = q.head
	q.len.Store(0)
	q.useWait = useWait
	if useWait {
		q.nextCh = make(chan bool)
		q.readValCh = make(chan bool)
		q.notifyCh = make(chan bool, 1)
		q.done = make(chan struct{})
		go q.startNotify()
	}
	return q
}

func (q *SafeQueue[T]) Enqueue(value T) {
	q.tail.AddLeft(value)
	if q.len.Add(1) == 1 && q.useWait {
		q.notifyCh <- true
	}
}

func (q *SafeQueue[T]) Dequeue() (T, bool) {
	if q.useWait {
		return q.Read(false)
	}
	defer q.len.Add(-1)
	return q.dequeue()
}

// Read blocks until a value is available in the queue, then returns it.
// This is similar to receiving from a channel.
// Setting wait to false will return immediately if no value is available.
// Only the first end channel is used, if provided. End channel is not used if wait is false.
func (q *SafeQueue[T]) Read(wait bool, end ...<-chan struct{}) (T, bool) {
	if !q.useWait {
		return q.Dequeue()
	}
	if wait {
		if len(end) == 0 {
			<-q.nextCh
			return q.readSuccess()
		} else {
			select {
			case <-q.nextCh:
				return q.readSuccess()
			case <-end[0]:
				var zero T
				return zero, false // Return zero value if end channel is closed
			}
		}
	} else {
		select {
		case <-q.nextCh:
			return q.readSuccess()
		default:
			var zero T
			return zero, false // No value available, return zero value
		}
	}
}

func (q *SafeQueue[T]) readSuccess() (T, bool) {
	defer func() {
		q.readValCh <- true // Notify that the value was read
	}()
	q.len.Add(-1) // Decrement length after reading
	return q.dequeue()
}

// helper to dequeue an item from the queue
func (q *SafeQueue[T]) dequeue() (T, bool) {
	ch, done := q.head.manageRightLocks()
	<-ch
	defer utils.SafeCloseChannel(done)
	n := q.head.next
	if n == q.tail {
		var zero T
		return zero, false // Queue is empty
	}
	chl, donel := n.manageRightLocks()
	<-chl
	defer utils.SafeCloseChannel(donel)
	value := n._RemoveSelf()
	return value, true
}

func (q *SafeQueue[T]) startNotify() {
	for {
		select {
		case <-q.done:
			return
		case <-q.notifyCh:
		}
		for q.len.Load() > 0 {
			select {
			case <-q.done:
				return
			default:
				q.nextCh <- true
				<-q.readValCh // Notify that the value was read
			}
		}
	}
}

func (q *SafeQueue[T]) Close() error {
	utils.SafeCloseChannel(q.done)
	utils.SafeCloseChannel(q.nextCh)
	utils.SafeCloseChannel(q.notifyCh)
	return nil
}

func (q *SafeQueue[T]) Size() int {
	return int(q.len.Load())
}
