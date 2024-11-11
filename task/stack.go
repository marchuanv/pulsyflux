package task

import (
	"slices"
	"sync"
)

type stack[T any] []T

var mu sync.Mutex

func (s stack[T]) Push(item T) stack[T] {
	defer mu.Unlock()
	mu.Lock()
	slices.Reverse(s)
	newStack := append(s, item)
	slices.Reverse(newStack)
	return newStack
}

func (s stack[T]) Pop() (T, stack[T]) {
	defer mu.Unlock()
	mu.Lock()
	l := len(s)
	var item T
	if l > 0 {
		item = s[0]
		slices.Reverse(s)
		newStack := s[:l-1]
		slices.Reverse(newStack)
		return item, newStack
	} else {
		return item, stack[T]{}
	}
}
