package sliceext

import (
	"sync"
)

type Stack[T any] interface {
	Len() int
	Push(item T)
	Pop() T
	Peek() T
	CloneLen() int
	ClonePush(item T)
	ClonePop() T
	ClonePeek() T
	CloneReset()
}

type stack[T any] struct {
	stk   []T
	mu    sync.Mutex
	clone *stack[T]
}

func NewStack[T any]() Stack[T] {
	st := &stack[T]{}
	st.CloneReset()
	return st
}

func (s *stack[T]) Len() int {
	return length(s)
}

func (s *stack[T]) Push(item T) {
	push(s.clone, item)
	push(s, item)
}

func (s *stack[T]) Pop() T {
	pop(s.clone)
	return pop(s)
}

func (s *stack[T]) Peek() T {
	return peek(s)
}

func (s *stack[T]) CloneLen() int {
	return length(s.clone)
}

func (s *stack[T]) ClonePush(item T) {
	push(s.clone, item)
}

func (s *stack[T]) CloneReset() {
	defer s.mu.Unlock()
	s.mu.Lock()
	s.clone = &stack[T]{}
	s.clone.stk = append(s.clone.stk, s.stk...)
}

func (s *stack[T]) ClonePop() T {
	return pop(s.clone)
}

func (s *stack[T]) ClonePeek() T {
	return peek(s.clone)
}

func length[T any](s *stack[T]) int {
	defer s.mu.Unlock()
	s.mu.Lock()
	return len(s.stk)
}

func push[T any](s *stack[T], item T) {
	defer s.mu.Unlock()
	s.mu.Lock()
	s.stk = append(s.stk, item)
}

func pop[T any](s *stack[T]) T {
	defer s.mu.Unlock()
	s.mu.Lock()
	var top T
	l := len(s.stk)
	if l > 0 {
		top = s.stk[l-1]
		s.stk = s.stk[:l-1]
	}
	return top
}

func peek[T any](s *stack[T]) T {
	defer s.mu.Unlock()
	s.mu.Lock()
	var top T
	l := len(s.stk)
	if l > 0 {
		top = s.stk[l-1]
	}
	return top
}
