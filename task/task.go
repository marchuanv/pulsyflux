package task

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

type task struct {
	errFunc         func(tsk *task)
	err             error
	errorParam      reflect.Value
	hasErrors       bool
	isErrorsHandled bool
	isChild         bool
	callstack       []string
}

var tskStack = taskStack{}
var mu = sync.Mutex{}

func Do[T1 any, T2 any](doFunc func() (T1, error), errorFunc ...func(err error, errorParam T2) T2) T1 {
	defer (func() {
		var recErr error
		r := recover()
		if r != nil {
			recErr = errors.New(fmt.Sprint(r))
		}
		if tskStack.Len() > 0 {
			popTsk := tskStack.Pop()

			if recErr != nil {
				popTsk.hasErrors = true
				popTsk.isErrorsHandled = false
				popTsk.err = recErr
			}

			nextTask := tskStack.Peek()
			isUnlocked := false
			if !popTsk.isChild {
				mu.Unlock()
				isUnlocked = true
			}
			popTsk.errFunc(popTsk)
			if nextTask != nil {
				nextTask.hasErrors = popTsk.hasErrors
				nextTask.isErrorsHandled = popTsk.isErrorsHandled
				nextTask.err = popTsk.err
				if !isZero(popTsk.errorParam) {

					nextTask.errorParam = popTsk.errorParam
				}
			}
			popTsk.callstack = nil
			popTsk.errorParam = reflect.Value{}
			popTsk.errFunc = nil
			if !popTsk.isChild {
				if popTsk.hasErrors && !popTsk.isErrorsHandled {
					panic(popTsk.err)
				}
				if !isUnlocked {
					mu.Unlock()
				}
			}
			popTsk.err = nil
		} else {
			panic(recErr)
		}
	})()
	errFunc := func(tsk *task) {
		tsk.isErrorsHandled = false
		if len(errorFunc) > 0 {
			if errorFunc[0] != nil {
				var errParam T2
				if !isZero(tsk.errorParam) {
					errParam = tsk.errorParam.Interface().(T2)
				}
				errParam = errorFunc[0](tsk.err, errParam)
				tsk.errorParam = reflect.ValueOf(errParam)
				tsk.isErrorsHandled = true
			}
		}
	}
	tsk := &task{
		errFunc,
		nil,
		reflect.Value{},
		false,
		true,
		false,
		[]string{},
	}
	skipFrame := 0
	for {
		caller := getFrame(skipFrame).Function
		if caller == "unknown" {
			break
		}
		tsk.callstack = append(tsk.callstack, caller)
		skipFrame += 1
	}
	if tskStack.Len() == 0 {
		mu.Lock()
		tsk.isChild = false
	} else {
		topTsk := tskStack.Peek()
		for _, clStk := range topTsk.callstack {
			for _, clStk2 := range tsk.callstack {
				if strings.Contains(clStk2, clStk) && !strings.Contains(clStk2, "task.Do") {
					tsk.isChild = true
				}
			}
		}
		if !tsk.isChild {
			mu.Lock()
		}
	}
	tskStack.Push(tsk)
	results, tskErr := doFunc()
	if tskErr != nil {
		tsk.hasErrors = true
		tsk.isErrorsHandled = false
		tsk.err = tskErr
	}
	return results
}

func isZero(val reflect.Value) bool {
	if val.IsValid() {
		if val.IsZero() {
			return true
		}
	} else {
		return true
	}
	return false
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
