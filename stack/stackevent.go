package stack

import (
	"sync"
	"time"
)

const (
	Push Event = iota
	Pop
	Enqueue
	Peek
)

type Event int

type StackEvent[T any] struct {
	stack                 map[Event][]func() (T, T)
	mu                    sync.Mutex
	invokeOnCreatedThread bool
}

func (se *StackEvent[T]) raise(e Event, current T, previous T) {
	funcStk := se.ensureReady(e)
	se.mu.Lock()
	funcStk = append(funcStk, func() (T, T) { return current, previous })
	se.stack[e] = funcStk
	se.mu.Unlock()
}

func (se *StackEvent[T]) subscribe(e Event, invokedByGoroutine bool) (T, T) {
	funcStk := se.ensureReady(e)
	se.mu.Lock()
	if invokedByGoroutine && !se.invokeOnCreatedThread {
		se.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return se.subscribe(e, invokedByGoroutine)
	}
	if !invokedByGoroutine {
		se.invokeOnCreatedThread = true
	}
	if len(funcStk) == 0 {
		se.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return se.subscribe(e, invokedByGoroutine)
	}
	eFunc := funcStk[len(funcStk)-1]
	se.stack[e] = funcStk[:len(funcStk)-1]
	se.mu.Unlock()
	return eFunc()
}

func (se *StackEvent[T]) subscribeAsync(e Event, on func(current T, previous T)) {
	go (func() {
		current, previous := se.subscribe(e, true)
		on(current, previous)
	})()
}

func (se *StackEvent[T]) ensureReady(e Event) []func() (T, T) {
	se.mu.Lock()
	if se.stack == nil {
		se.stack = make(map[Event][]func() (T, T))
	}
	funcStk, exists := se.stack[e]
	if !exists {
		se.stack[e] = []func() (T, T){}
	}
	se.mu.Unlock()
	return funcStk
}
