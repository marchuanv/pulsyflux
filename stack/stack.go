package stack

import (
	"slices"
	"sync"
)

type Stack[T any] struct {
	stack []T
	mu    sync.Mutex
}

func (s *Stack[T]) Len() int {
	defer s.mu.Unlock()
	s.mu.Lock()
	l := len(s.stack)
	return l
}

func (s *Stack[T]) Push(item T) {
	defer s.mu.Unlock()
	s.mu.Lock()
	s.stack = append(s.stack, item)
}

func (s *Stack[T]) Enqueue(item T) {
	defer s.mu.Unlock()
	s.mu.Lock()
	slices.Reverse(s.stack)
	s.stack = append(s.stack, item)
	slices.Reverse(s.stack)
}

func (s *Stack[T]) Pop() T {
	var top T
	defer s.mu.Unlock()
	s.mu.Lock()
	l := len(s.stack)
	if l > 0 {
		top = s.stack[l-1]
		s.stack = s.stack[:l-1]
	}
	return top
}

func (s *Stack[T]) Peek() T {
	defer s.mu.Unlock()
	s.mu.Lock()
	var top T
	l := len(s.stack)
	if l > 0 {
		top = s.stack[l-1]
	}
	return top
}
