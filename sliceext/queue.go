package sliceext

import (
	"slices"
	"sync"
)

type Queue[T any] struct {
	queue []T
	mu    sync.Mutex
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

func (s *Queue[T]) Len() int {
	defer s.mu.Unlock()
	s.mu.Lock()
	l := len(s.queue)
	return l
}

func (s *Queue[T]) Enqueue(item T) {
	defer s.mu.Unlock()
	s.mu.Lock()
	slices.Reverse(s.queue)
	s.queue = append(s.queue, item)
	slices.Reverse(s.queue)
}

func (q *Queue[T]) Dequeue() T {
	var front T
	defer q.mu.Unlock()
	q.mu.Lock()
	l := len(q.queue)
	if l > 0 {
		front = q.queue[l-1]
		q.queue = q.queue[:l-1]
	}
	return front
}

func (q *Queue[T]) Peek() T {
	defer q.mu.Unlock()
	q.mu.Lock()
	var front T
	l := len(q.queue)
	if l > 0 {
		front = q.queue[l-1]
	}
	return front
}
