package sliceext

import (
	"slices"
	"sync"
)

type slice[T comparable] struct {
	arr []T
	mu  sync.Mutex
}

func newSlice[T comparable]() *slice[T] {
	return &slice[T]{}
}

func (l *slice[T]) len() int {
	defer l.mu.Unlock()
	l.mu.Lock()
	return len(l.arr)
}

func (l *slice[T]) peek() T {
	defer l.mu.Unlock()
	l.mu.Lock()
	var item T
	length := len(l.arr)
	if length > 0 {
		item = l.arr[length-1]
	}
	return item
}

func (l *slice[T]) append(item T) {
	defer l.mu.Unlock()
	l.mu.Lock()
	l.arr = append(l.arr, item)
}

func (l *slice[T]) reverse() {
	defer l.mu.Unlock()
	l.mu.Lock()
	slices.Reverse(l.arr)
}

func (l *slice[T]) remove() T {
	defer l.mu.Unlock()
	l.mu.Lock()
	var item T
	length := len(l.arr)
	if length > 0 {
		item = l.arr[length-1]
		l.arr = l.arr[:length-1]
	}
	return item
}

func (l *slice[T]) getAt(index int) T {
	defer l.mu.Unlock()
	l.mu.Lock()
	var item T
	length := len(l.arr)
	if length > 0 {
		item = l.arr[index]
	}
	return item
}

func (l *slice[T]) rmvAt(index int) {
	defer l.mu.Unlock()
	l.mu.Lock()
	l.arr = append(l.arr[:index], l.arr[index+1:]...)
}
