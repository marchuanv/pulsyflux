package task

import (
	"fmt"
	"runtime"
	"slices"
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
	var caller string
	var callerStk = buildCallerStack()
	caller, callerStk = callerStk.Pop()
	tskClStk := &tskCallstack{caller, tsk, sync.Mutex{}}
	tskClStk.mu.Lock()
	fmt.Printf("\r\nattempting to push task for '%s' to the call stack\r\n", tskClStk.caller)
	topOfStack := tskClStks.Peek()
	if topOfStack == nil {
		tskClStks = tskClStks.Push(tskClStk)
		fmt.Printf("pushed task for '%s' to the call stack\r\n", tskClStk.caller)
	} else if strings.Contains(caller, topOfStack.caller) {
		tskClStks = tskClStks.Push(tskClStk)
		fmt.Printf("pushed task for '%s' to the call stack\r\n", tskClStk.caller)
	} else {
		var foundClStk *tskCallstack
		for len(callerStk) > 0 && foundClStk == nil {
			for _, tClStk := range tskClStks {
				if strings.Contains(caller, tClStk.caller) {
					foundClStk = tClStk
				}
			}
			caller, callerStk = callerStk.Pop()
		}
		if foundClStk == nil {
			go (func() {
				obtainedLock := foundClStk.mu.TryLock()
				for !obtainedLock {
					time.Sleep(1 * time.Second)
					obtainedLock = foundClStk.mu.TryLock()
				}
				foundClStk.mu.Lock()
				tskClStks = tskClStks.Push(tskClStk)
				fmt.Printf("pushed task for '%s' to the call stack\r\n", tskClStk.caller)
				foundClStk.mu.Unlock()
			})()
		} else {
			tskClStks = tskClStks.Push(tskClStk)
			fmt.Printf("pushed task for '%s' to the call stack\r\n", tskClStk.caller)
		}
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

func buildCallerStack() stack[string] {
	var callers []string
	skipFrame := 2
	caller := getFrame(skipFrame).Function
	for caller != "unknown" {
		if !strings.Contains(caller, "pulsyflux/task.Do[...]") {
			callers = append(callers, caller)
		}
		skipFrame += 1
		caller = getFrame(skipFrame).Function
	}
	slices.Reverse(callers)
	var callerStk = stack[string]{}
	for _, caller := range callers {
		callerStk = callerStk.Push(caller)
	}
	return callerStk
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
