package stack

import (
	"slices"
	"sync"
)

type Stack[T any] struct {
	stack  []T
	events *StackEvent[T]
}

var mu sync.Mutex

func (s *Stack[T]) On(e Event) (T, T) {
	s.initialise()
	return s.events.subscribe(e)
}

func (s *Stack[T]) OnAsync(e Event, f func(current T, previous T)) {
	s.initialise()
	s.events.subscribeAsync(e, f)
}

func (s *Stack[T]) Len() int {
	defer mu.Unlock()
	mu.Lock()
	return len(s.stack)
}

func (s *Stack[T]) Push(item T) {
	defer mu.Unlock()
	mu.Lock()
	l := len(s.stack)
	var previous T
	if l > 0 {
		previous = s.stack[l-1]
	}
	s.stack = append(s.stack, item)
	s.initialise()
	s.events.raise(Push, item, previous)
}

func (s *Stack[T]) Enqueue(item T) {
	defer mu.Unlock()
	mu.Lock()
	var previous T
	slices.Reverse(s.stack)
	l := len(s.stack)
	if l > 0 {
		previous = s.stack[l-1]
	}
	s.stack = append(s.stack, item)
	slices.Reverse(s.stack)
	s.initialise()
	s.events.raise(Enqueue, item, previous)
}

func (s *Stack[T]) Pop() {
	defer mu.Unlock()
	mu.Lock()
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
	s.initialise()
	s.events.raise(Pop, current, previous)
}

func (s *Stack[T]) Peek() {
	defer mu.Unlock()
	mu.Lock()
	var current T
	var previous T
	l := len(s.stack)
	if l > 0 {
		current = s.stack[l-1]
		if l > 1 {
			previous = s.stack[l-2]
		}
	}
	s.initialise()
	s.events.raise(Peek, current, previous)
}

func (s *Stack[T]) initialise() {
	if s.events == nil {
		s.events = &StackEvent[T]{make(chan func() (Event, T, T)), false}
		go s.events.initialise()
	}
}
