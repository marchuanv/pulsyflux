package task

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

type tskCallstack struct {
	caller string
	task   *task
	mu     sync.Mutex
}

var tskClStks = stack[*tskCallstack]{}

func callstackPush(tsk *task) {
	skipFrame := 2
	caller := getFrame(skipFrame).Function
	if caller == "unknown" {
		panic("Do stack error")
	}
	tskClStk := &tskCallstack{caller, tsk, sync.Mutex{}}
	tskClStk.mu.Lock()
	fmt.Printf("\r\nattempting to push task for '%s' to the call stack\r\n", tskClStk.caller)
	var topOfStack *tskCallstack
	if len(tskClStks) > 0 {
		topOfStack = tskClStks[0]
	}
	if topOfStack == nil {
		tskClStks = tskClStks.Push(tskClStk)
		fmt.Printf("pushed task for '%s' to the call stack\r\n", tskClStk.caller)
	} else if strings.Contains(caller, topOfStack.caller) {
		tskClStks = tskClStks.Push(tskClStk)
		fmt.Printf("pushed task for '%s' to the call stack\r\n", tskClStk.caller)
	} else {
		go (func() {
			obtainedLock := topOfStack.mu.TryLock()
			for !obtainedLock {
				time.Sleep(1 * time.Second)
				obtainedLock = topOfStack.mu.TryLock()
			}
			topOfStack.mu.Lock()
			tskClStks = tskClStks.Push(tskClStk)
			fmt.Printf("pushed task for '%s' to the call stack\r\n", tskClStk.caller)
			topOfStack.mu.Unlock()
		})()
	}
}

func callstackPop() *task {
	if len(tskClStks) > 0 {
		var tskClStk *tskCallstack
		tskClStk, tskClStks = tskClStks.Pop()
		fmt.Printf("\r\npopped task for '%s' from the call stack\r\n", tskClStk.caller)
		tskClStk.mu.Unlock()
		return tskClStk.task
	} else {
		return nil
	}
}

func callstackPeek() *task {
	if len(tskClStks) > 0 {
		return tskClStks[0].task
	} else {
		return nil
	}
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
