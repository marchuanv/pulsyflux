package sliceext

import (
	"sync"

	"github.com/google/uuid"
)

type Stack[T any] interface {
	Len() int
	Push(item T)
	Pop() T
	Peek() T
	Clone(cloneId uuid.UUID)
	CloneLen(cloneId uuid.UUID) int
	ClonePop(cloneId uuid.UUID) T
	ClonePeek(cloneId uuid.UUID) T
	CloneClear(cloneId uuid.UUID)
}

type stack[T any] struct {
	stk      []T
	clones   map[uuid.UUID]*stack[T]
	stkMu    sync.Mutex
	clonesMu sync.Mutex
}

func NewStack[T any]() Stack[T] {
	st := &stack[T]{}
	st.clones = make(map[uuid.UUID]*stack[T])
	return st
}

func (s *stack[T]) Len() int {
	return stkLen(s)
}

func (s *stack[T]) Push(item T) {
	defer s.clonesMu.Unlock()
	s.clonesMu.Lock()
	for _, cloneStk := range s.clones {
		stkPush(cloneStk, item)
	}
	stkPush(s, item)
}

func (s *stack[T]) Pop() T {
	return stkPop(s)
}

func (s *stack[T]) Peek() T {
	return stkPeek(s)
}

func (s *stack[T]) CloneLen(cloneId uuid.UUID) int {
	defer s.clonesMu.Unlock()
	s.clonesMu.Lock()
	cloneStk, exists := s.clones[cloneId]
	if exists {
		return stkLen(cloneStk)
	}
	return 0
}

func (s *stack[T]) Clone(cloneId uuid.UUID) {
	defer (func() {
		s.stkMu.Unlock()
		s.clonesMu.Unlock()
	})()
	s.stkMu.Lock()
	s.clonesMu.Lock()
	delete(s.clones, cloneId)
	clnStk := NewStack[T]().(*stack[T])
	s.clones[cloneId] = clnStk
	clnStk.stk = append(clnStk.stk, s.stk...)
}

func (s *stack[T]) CloneClear(cloneId uuid.UUID) {
	defer s.clonesMu.Unlock()
	s.clonesMu.Lock()
	delete(s.clones, cloneId)
}

func (s *stack[T]) ClonePop(cloneId uuid.UUID) T {
	defer s.clonesMu.Unlock()
	s.clonesMu.Lock()
	var popped T
	cloneStk, exists := s.clones[cloneId]
	if exists {
		popped = stkPop(cloneStk)
	}
	return popped
}

func (s *stack[T]) ClonePeek(cloneId uuid.UUID) T {
	defer s.clonesMu.Unlock()
	s.clonesMu.Lock()
	var pk T
	cloneStk, exists := s.clones[cloneId]
	if exists {
		pk = stkPeek(cloneStk)
	}
	return pk
}

func stkLen[T any](s *stack[T]) int {
	defer s.stkMu.Unlock()
	s.stkMu.Lock()
	return len(s.stk)
}

func stkPush[T any](s *stack[T], item T) {
	defer s.stkMu.Unlock()
	s.stkMu.Lock()
	s.stk = append(s.stk, item)
}

func stkPop[T any](s *stack[T]) T {
	defer s.stkMu.Unlock()
	s.stkMu.Lock()
	var top T
	l := len(s.stk)
	if l > 0 {
		top = s.stk[l-1]
		s.stk = s.stk[:l-1]
	}
	return top
}

func stkPeek[T any](s *stack[T]) T {
	defer s.stkMu.Unlock()
	s.stkMu.Lock()
	var top T
	l := len(s.stk)
	if l > 0 {
		top = s.stk[l-1]
	}
	return top
}
