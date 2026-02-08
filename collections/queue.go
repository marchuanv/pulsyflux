package collections

import "sync"

type Queue[T any] struct {
	mu    sync.Mutex
	items []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		items: make([]T, 0),
	}
}

func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *Queue[T]) Enqueue(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
}

func (q *Queue[T]) Dequeue() T {
	q.mu.Lock()
	defer q.mu.Unlock()
	var item T
	if len(q.items) > 0 {
		item = q.items[0]
		q.items = q.items[1:]
	}
	return item
}

func (q *Queue[T]) Peek() T {
	q.mu.Lock()
	defer q.mu.Unlock()
	var item T
	if len(q.items) > 0 {
		item = q.items[0]
	}
	return item
}
