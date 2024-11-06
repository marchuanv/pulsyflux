package task

import (
	"errors"
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
	var results T1
	errFunc := func(tsk *task) {
		tsk.isErrorsHandled = false
		if len(errorFunc) > 0 {
			if errorFunc[0] != nil {
				var errParam T2
				if !isZero(tsk.errorParam) {
					errParam = getValue(tsk.errorParam).Interface().(T2)
				}
				errParam = errorFunc[0](tsk.err, errParam)
				tsk.errorParam = getValue(reflect.ValueOf(errParam))
				tsk.isErrorsHandled = true
			}
		}
	}
	{
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
		res, tskErr := doFunc()
		if tskErr != nil {
			tsk.hasErrors = true
			tsk.isErrorsHandled = false
			tsk.err = tskErr
		}
		if tsk.isChild {
			return res
		} else {
			results = res
		}
	}
	var popTsk *task
	var prevPopTsk *task

	undisposedTskStack := taskStack{}
	for tskStack.Len() > 0 {
		popTsk = tskStack.Pop()
		if prevPopTsk == nil {
			prevPopTsk = popTsk
		}
		popTsk.hasErrors = prevPopTsk.hasErrors
		popTsk.isErrorsHandled = prevPopTsk.isErrorsHandled
		popTsk.err = prevPopTsk.err
		if isZero(popTsk.errorParam) {
			if !isZero(prevPopTsk.errorParam) {
				popTsk.errorParam = prevPopTsk.errorParam
			}
		} else {
			if !isZero(prevPopTsk.errorParam) {
				if !popTsk.errorParam.Type().AssignableTo(prevPopTsk.errorParam.Type()) {
					unassignableErr := errors.New("errorParamVal is not assignable")
					panic(unassignableErr)
				}
				popTsk.errorParam = prevPopTsk.errorParam
			}
		}
		popTsk.errFunc(popTsk)
		undisposedTskStack.Push(popTsk)
		prevPopTsk = popTsk
	}
	if popTsk.hasErrors && !popTsk.isErrorsHandled {
		tskErr := errors.New("one or more unhandled errors occured in the task chain ")
		panic(tskErr)
	}
	//cleanup
	for undisposedTskStack.Len() > 0 {
		undTskSt := undisposedTskStack.Pop()
		if !isZero(undTskSt.errorParam) {
			undTskSt.callstack = nil
			undTskSt.errorParam = reflect.Value{}
			undTskSt.errFunc = nil
			undTskSt.err = nil
		}
	}
	mu.Unlock()
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

func getValue(val reflect.Value) reflect.Value {
	return reflect.Indirect(val)
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
