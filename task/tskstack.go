package task

import "sync"

type taskStack struct {
	tasks  []*task
	mu     sync.Mutex
	locked bool
}

func (s *taskStack) Push(tsk *task) {
	s.tasks = append(s.tasks, tsk)
}

func (s *taskStack) Pop() *task {
	l := len(s.tasks)
	tsk := s.tasks[l-1]
	s.tasks = s.tasks[:l-1]
	return tsk
}

func (s *taskStack) Peek() *task {
	l := len(s.tasks)
	return s.tasks[l-1]
}

func (s *taskStack) Lock() {
	s.mu.Lock()
}

func (s *taskStack) Unlock() {
	s.mu.Unlock()
}

func (s *taskStack) IsLocked() bool {
	return !s.locked
}

func (s *taskStack) Len() int {
	return len(s.tasks)
}
