package stack

import (
	"slices"
	"sync"
)

type Stack[T any] struct {
	stack  []T
	events *StackEvent[T]
	mu     sync.Mutex
}

func (s *Stack[T]) On(e Event) (T, T) {
	s.initialise()
	return s.events.subscribe(e, false)
}

func (s *Stack[T]) OnAsync(e Event, f func(current T, previous T)) {
	s.mu.Lock()
	s.initialise()
	s.mu.Unlock()
	s.events.subscribeAsync(e, f)
}

func (s *Stack[T]) Len() int {
	s.mu.Lock()
	l := len(s.stack)
	s.mu.Unlock()
	return l
}

func (s *Stack[T]) Push(item T) {
	s.mu.Lock()
	s.initialise()
	l := len(s.stack)
	var previous T
	if l > 0 {
		previous = s.stack[l-1]
	}
	s.stack = append(s.stack, item)
	s.events.raise(Push, item, previous)
	s.mu.Unlock()
}

func (s *Stack[T]) Enqueue(item T) {
	defer s.mu.Unlock()
	s.mu.Lock()
	s.initialise()
	var previous T
	slices.Reverse(s.stack)
	l := len(s.stack)
	if l > 0 {
		previous = s.stack[l-1]
	}
	s.stack = append(s.stack, item)
	slices.Reverse(s.stack)
	s.events.raise(Enqueue, item, previous)
}

func (s *Stack[T]) Pop() {
	defer s.mu.Unlock()
	s.mu.Lock()
	s.initialise()
	l := len(s.stack)
	var current T
	var previous T
	if l > 0 {
		previous = s.stack[l-1]
		s.stack = s.stack[:l-1]
		l := len(s.stack)
		if l > 0 {
			current = s.stack[l-1]
		}
	}
	s.events.raise(Pop, current, previous)
}

func (s *Stack[T]) Peek() {
	s.mu.Lock()
	s.initialise()
	var current T
	var previous T
	l := len(s.stack)
	if l > 0 {
		current = s.stack[l-1]
		if l > 1 {
			previous = s.stack[l-2]
		}
	}
	s.events.raise(Peek, current, previous)
	s.mu.Unlock()
}

func (s *Stack[T]) initialise() {
	if s.events == nil {
		s.events = &StackEvent[T]{}
	}
}
