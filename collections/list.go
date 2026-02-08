package collections

import "sync"

type List[T comparable] struct {
	mu    sync.RWMutex
	items []T
}

func NewList[T comparable]() *List[T] {
	return &List[T]{
		items: make([]T, 0),
	}
}

func (l *List[T]) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.items)
}

func (l *List[T]) Has(item T) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, v := range l.items {
		if v == item {
			return true
		}
	}
	return false
}

func (l *List[T]) All() []T {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]T, len(l.items))
	copy(result, l.items)
	return result
}

func (l *List[T]) Delete(item T) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, v := range l.items {
		if v == item {
			l.items = append(l.items[:i], l.items[i+1:]...)
			return true
		}
	}
	return false
}

func (l *List[T]) Add(item T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, item)
}

func (l *List[T]) Clone() *List[T] {
	l.mu.RLock()
	defer l.mu.RUnlock()
	clone := NewList[T]()
	clone.items = make([]T, len(l.items))
	copy(clone.items, l.items)
	return clone
}

func (l *List[T]) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = make([]T, 0)
}
