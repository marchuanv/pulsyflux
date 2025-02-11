package sliceext

type Queue[T comparable] struct {
	slice *slice[T]
}

func NewQueue[T comparable]() *Queue[T] {
	return &Queue[T]{newSlice[T]()}
}

func (q *Queue[T]) Len() int {
	return q.slice.len()
}

func (q *Queue[T]) Enqueue(item T) {
	q.slice.append(item)
}

func (q *Queue[T]) Dequeue() T {
	var item T
	if q.slice.len() > 0 {
		q.slice.reverse()
		item = q.slice.remove()
		q.slice.reverse()
	}
	return item
}

func (q *Queue[T]) Peek() T {
	var item T
	if q.slice.len() > 0 {
		q.slice.reverse()
		item = q.slice.peek()
		q.slice.reverse()
	}
	return item
}
