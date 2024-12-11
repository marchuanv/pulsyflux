package sliceext

import (
	"slices"
	"sync"

	"github.com/google/uuid"
)

type Queue[T any] interface {
	Len() int
	Enqueue(item T)
	Dequeue() T
	Peek() T
	Clone(cloneId uuid.UUID)
	CloneLen(cloneId uuid.UUID) int
	CloneDequeue(cloneId uuid.UUID) T
	ClonePeek(cloneId uuid.UUID) T
	CloneClear(cloneId uuid.UUID)
}

type queue[T any] struct {
	q        []T
	clones   map[uuid.UUID]*queue[T]
	qMu      sync.Mutex
	clonesMu sync.Mutex
}

func NewQueue[T any]() Queue[T] {
	q := &queue[T]{}
	q.clones = make(map[uuid.UUID]*queue[T])
	return q
}

func (q *queue[T]) Len() int {
	return qLen(q)
}

func (q *queue[T]) Enqueue(item T) {
	defer q.clonesMu.Unlock()
	q.clonesMu.Lock()
	for _, cloneQ := range q.clones {
		qEnqueue(cloneQ, item)
	}
	qEnqueue(q, item)
}

func (q *queue[T]) Dequeue() T {
	return qDequeue(q)
}

func (q *queue[T]) Peek() T {
	return qPeek(q)
}

func (q *queue[T]) CloneLen(cloneId uuid.UUID) int {
	defer q.clonesMu.Unlock()
	q.clonesMu.Lock()
	cloneQ, exists := q.clones[cloneId]
	if exists {
		return qLen(cloneQ)
	}
	return 0
}

func (q *queue[T]) Clone(cloneId uuid.UUID) {
	defer (func() {
		q.qMu.Unlock()
		q.clonesMu.Unlock()
	})()
	q.qMu.Lock()
	q.clonesMu.Lock()
	delete(q.clones, cloneId)
	clnQ := NewQueue[T]().(*queue[T])
	q.clones[cloneId] = clnQ
	clnQ.q = append(clnQ.q, q.q...)
}

func (q *queue[T]) CloneClear(cloneId uuid.UUID) {
	defer q.clonesMu.Unlock()
	q.clonesMu.Lock()
	delete(q.clones, cloneId)
}

func (q *queue[T]) CloneDequeue(cloneId uuid.UUID) T {
	defer q.clonesMu.Unlock()
	q.clonesMu.Lock()
	var dequeued T
	cloneQ, exists := q.clones[cloneId]
	if exists {
		dequeued = qDequeue(cloneQ)
	}
	return dequeued
}

func (q *queue[T]) ClonePeek(cloneId uuid.UUID) T {
	defer q.clonesMu.Unlock()
	q.clonesMu.Lock()
	var pk T
	cloneQ, exists := q.clones[cloneId]
	if exists {
		pk = qPeek(cloneQ)
	}
	return pk
}

func qLen[T any](q *queue[T]) int {
	defer q.qMu.Unlock()
	q.qMu.Lock()
	return len(q.q)
}

func qEnqueue[T any](q *queue[T], item T) {
	defer q.qMu.Unlock()
	q.qMu.Lock()
	q.q = append(q.q, item)
}

func qDequeue[T any](q *queue[T]) T {
	defer q.qMu.Unlock()
	q.qMu.Lock()
	var first T
	l := len(q.q)
	if l > 0 {
		first = q.q[0]
		slices.Reverse(q.q)
		q.q = q.q[:l-1]
		slices.Reverse(q.q)
	}
	return first
}

func qPeek[T any](q *queue[T]) T {
	defer q.qMu.Unlock()
	q.qMu.Lock()
	var first T
	l := len(q.q)
	if l > 0 {
		first = q.q[0]
	}
	return first
}
