package sliceext

import (
	"slices"
	"sync"
)

type slice[T any] struct {
	arr []T
	mu  sync.Mutex
}

func newSlice[T any]() *slice[T] {
	return &slice[T]{}
}

func (s *slice[T]) len() int {
	defer s.mu.Unlock()
	s.mu.Lock()
	return len(s.arr)
}

func (s *slice[T]) peek() T {
	defer s.mu.Unlock()
	s.mu.Lock()
	var item T
	length := len(s.arr)
	if length > 0 {
		item = s.arr[length-1]
	}
	return item
}

func (s *slice[T]) append(item T) {
	defer s.mu.Unlock()
	s.mu.Lock()
	s.arr = append(s.arr, item)
}

func (s *slice[T]) reverse() {
	defer s.mu.Unlock()
	s.mu.Lock()
	slices.Reverse(s.arr)
}

func (s *slice[T]) remove() T {
	defer s.mu.Unlock()
	s.mu.Lock()
	var item T
	length := len(s.arr)
	if length > 0 {
		item = s.arr[length-1]
		s.arr = s.arr[:length-1]
	}
	return item
}

func (s *slice[T]) getAt(index int) T {
	defer s.mu.Unlock()
	s.mu.Lock()
	var item T
	length := len(s.arr)
	if length > 0 {
		item = s.arr[index]
	}
	return item
}

func (s *slice[T]) rmvAt(index int) {
	defer s.mu.Unlock()
	s.mu.Lock()
	s.arr = append(s.arr[:index], s.arr[index+1:]...)
}

func (s *slice[T]) copy() *slice[T] {
	sliceCln := newSlice[T]()
	sliceCln.arr = make([]T, len(s.arr))
	copy(sliceCln.arr, s.arr)
	return sliceCln
}
