package task

import (
	"runtime"
	"strings"
	"sync"
)

type tskCallstack struct {
	caller string
	task   *task
}

var callstackMu sync.Mutex
var callstack = stack[*tskCallstack]{}

func callstackPush(tsk *task) {
	skipFrame := 2
	caller := getFrame(skipFrame).Function
	if caller == "unknown" {
		panic("Do stack error")
	}
	existingTask := getTaskInCallstack(caller)
	if existingTask != nil {
		go (func() {
			relatedMu.Lock()
			if existingTask.isCalled {
				relatedMu.Unlock()
				callstack = callstack.Push(&tskCallstack{caller, tsk})
			} else {
				callstack = callstack.Push(&tskCallstack{caller, tsk})
			}
		})()
	} else if len(callstack) == 0 {
		callstackMu.Lock()
		callstack = callstack.Push(&tskCallstack{caller, tsk})
	} else {
		callstackMu.Lock()
		callstack = callstack.Push(&tskCallstack{caller, tsk})
	}
}

func callstackPop() (string, *task) {
	defer callstackMu.Unlock()
	callstackMu.Lock()
	var tskClStk *tskCallstack
	var nextTskClStk1 *tskCallstack
	var nextTskClStk2 *tskCallstack
	var requeuedtskClStk *tskCallstack
	for {
		nextTskClStk1, callstack = callstack.Pop()
		if nextTskClStk1 != nil {
			if tskClStk != nil && nextTskClStk1.task == tskClStk.task {
				continue
			}
			if nextTskClStk1 == requeuedtskClStk { //reached the end
				break
			}
			nextTskClStk2, callstack = callstack.Pop()
			if nextTskClStk2 != nil {
				if nextTskClStk2.task == nextTskClStk1.task {
					tskClStk = nextTskClStk1
				} else {
					callstack = callstack.Enqueue(nextTskClStk2)
					requeuedtskClStk = nextTskClStk2
				}
			} else {
				break
			}
		} else {
			break
		}
	}
	return tskClStk.caller, tskClStk.task
}

func callstackPeek() (string, *task) {
	defer callstackMu.Unlock()
	callstackMu.Lock()
	return callstack[0].caller, callstack[0].task
}

func getTaskInCallstack(caller string) *task {
	for _, clstk := range callstack {
		if strings.Contains(caller, clstk.caller) {
			return clstk.task
		}
	}
	return nil
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
