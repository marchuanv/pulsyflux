package task

import (
	"errors"
	"fmt"
	"reflect"
)

type task struct {
	err             error
	errorParam      reflect.Value
	hasErrors       bool
	isErrorsHandled bool
}

var tskStack = stack[*task]{}

func Do[T1 any, T2 any](doFunc func() (T1, error), receive func(variable T1), errorFuncs ...func(err error, errorParam T2) T2) T1 {
	var results T1
	execute := func() T1 {
		var tsk *task
		var nextTask *task
		tsk, tskStack = tskStack.Pop()
		nextTask, tskStack = tskStack.Pop()
		var errorFunc = errorFuncs[0]
		results = runTask(doFunc, tsk)
		if tsk.hasErrors {
			runTaskError(errorFunc, tsk)
			if nextTask == nil {
				if tsk.hasErrors && !tsk.isErrorsHandled {
					panic(tsk.err)
				}
			} else {
				nextTask.hasErrors = tsk.hasErrors
				nextTask.isErrorsHandled = tsk.isErrorsHandled
				nextTask.err = tsk.err
				if !isZero(tsk.errorParam) {
					nextTask.errorParam = tsk.errorParam
				}
			}
			tsk.errorParam = reflect.Value{}
			tsk.err = nil
		} else if receive != nil {
			receive(results)
		}
		return results
	}
	tskStack = tskStack.Push(&task{
		nil,
		reflect.Value{},
		false,
		true,
	})
	if receive == nil {
		results = execute()
	} else {
		go execute()
	}
	return results
}

func runTask[T1 any](doFunc func() (T1, error), tsk *task) T1 {
	defer (func() {
		var recErr error
		r := recover()
		if r != nil {
			recErr = errors.New(fmt.Sprint(r))
		}
		if recErr != nil {
			tsk.hasErrors = true
			tsk.isErrorsHandled = false
			tsk.err = recErr
		}
	})()
	results, tskErr := doFunc()
	if tskErr != nil {
		tsk.hasErrors = true
		tsk.isErrorsHandled = false
		tsk.err = tskErr
	}
	return results
}

func runTaskError[T any](errorFunc func(err error, errorParam T) T, tsk *task) {
	defer (func() {
		var recErr error
		r := recover()
		if r != nil {
			recErr = errors.New(fmt.Sprint(r))
		}
		if recErr != nil {
			tsk.hasErrors = true
			tsk.isErrorsHandled = false
			tsk.err = recErr
		}
	})()
	tsk.isErrorsHandled = false
	var errParam T
	if !isZero(tsk.errorParam) {
		errParam = tsk.errorParam.Interface().(T)
	}
	errParam = errorFunc(tsk.err, errParam)
	tsk.errorParam = reflect.ValueOf(errParam)
	tsk.isErrorsHandled = true
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
