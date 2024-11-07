package task

import (
	"runtime"
	"strings"
	"sync"
)

type taskStack struct {
	tasks     []*task
	mu        sync.Mutex
	callstack []string
	isLocked  bool
}

func (s *taskStack) Push(tsk *task) {
	skipFrame := 0
	callstack := []string{}
	for {
		caller := getFrame(skipFrame).Function
		if caller == "unknown" {
			break
		}
		if !strings.Contains(caller, "task.(*taskStack).Push") && !strings.Contains(caller, "task.Do") {
			callstack = append(callstack, caller)
		}
		skipFrame += 1
	}
	if !s.isLocked {
		s.mu.Lock()
		s.isLocked = true
		s.tasks = append(s.tasks, tsk)
		s.callstack = callstack //reset
	} else {
		allow := false
		for _, clStk := range s.callstack {
			for _, clStk2 := range callstack {
				if strings.Contains(clStk2, clStk) {
					allow = true
					s.callstack = append(s.callstack, clStk2)
					break
				}
			}
		}
		if allow {
			s.tasks = append(s.tasks, tsk)
		} else {
			s.mu.Lock()
			s.tasks = append(s.tasks, tsk)
			s.callstack = callstack
		}
	}
}

func (s *taskStack) Pop() *task {
	l := len(s.tasks)
	if l > 0 {
		tsk := s.tasks[l-1]
		s.tasks = s.tasks[:l-1]
		return tsk
	} else {
		s.mu.Unlock()
		s.isLocked = false
		return nil
	}
}

func (s *taskStack) Peek() *task {
	l := len(s.tasks)
	if l > 0 {
		return s.tasks[l-1]
	} else {
		return nil
	}
}

func (s *taskStack) Len() int {
	return len(s.tasks)
}

func getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2
	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)
	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}
	return frame
}
