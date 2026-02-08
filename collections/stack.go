package collections

import "sync"

type Stack[T any] struct {
	mu    sync.Mutex
	items []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{
		items: make([]T, 0),
	}
}

func (s *Stack[T]) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

func (s *Stack[T]) Push(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	var item T
	if len(s.items) > 0 {
		item = s.items[len(s.items)-1]
		s.items = s.items[:len(s.items)-1]
	}
	return item
}

func (s *Stack[T]) Peek() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	var item T
	if len(s.items) > 0 {
		item = s.items[len(s.items)-1]
	}
	return item
}

func (s *Stack[T]) Clone() *Stack[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := NewStack[T]()
	clone.items = make([]T, len(s.items))
	copy(clone.items, s.items)
	return clone
}

func (s *Stack[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make([]T, 0)
}
