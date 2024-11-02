package task

import "sync"

type taskStack []*task

var mu = sync.Mutex{}

func (s taskStack) Push(tsk *task) taskStack {
	defer mu.Unlock()
	mu.Lock()
	return append(s, tsk)
}

func (s taskStack) Pop() (taskStack, *task) {
	defer mu.Unlock()
	mu.Lock()
	l := len(s)
	return s[:l-1], s[l-1]
}

func (s taskStack) Peek() *task {
	defer mu.Unlock()
	mu.Lock()
	l := len(s)
	return s[l-1]
}
