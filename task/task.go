package task

import (
	"errors"
	"fmt"
	"reflect"
)

type task struct {
	errFunc         func(tsk *task)
	err             error
	errorParam      reflect.Value
	hasErrors       bool
	isErrorsHandled bool
}

var tskStack = taskStack{}

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
			popTsk.errFunc(popTsk)
			nextTask := tskStack.Peek()
			if nextTask == nil {
				if popTsk.hasErrors && !popTsk.isErrorsHandled {
					panic(popTsk.err)
				}
			} else {
				nextTask.hasErrors = popTsk.hasErrors
				nextTask.isErrorsHandled = popTsk.isErrorsHandled
				nextTask.err = popTsk.err
				if !isZero(popTsk.errorParam) {
					nextTask.errorParam = popTsk.errorParam
				}
			}
			popTsk.errorParam = reflect.Value{}
			popTsk.errFunc = nil
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
