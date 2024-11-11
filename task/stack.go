package task

import (
	"sync"
)

type stack[T any] []T

var mu sync.Mutex

func (s stack[T]) Push(item T) stack[T] {
	defer mu.Unlock()
	mu.Lock()
	return append(s, item)
}

func (s stack[T]) Pop() (T, stack[T]) {
	defer mu.Unlock()
	mu.Lock()
	l := len(s)
	var item T
	if l > 0 {
		item = s[l-1]
		newStack := s[:l-1]
		return item, newStack
	} else {
		return item, stack[T]{}
	}
}

func (s stack[T]) Peek() T {
	defer mu.Unlock()
	mu.Lock()
	var item T
	l := len(s)
	if l > 0 {
		item = s[l-1]
	}
	return item
}
