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

func Do[T1 any, T2 any](doFunc func() (T1, error), errorFunc ...func(err error, errorParam T2) T2) T1 {
	defer (func() {
		var recErr error
		r := recover()
		if r != nil {
			recErr = errors.New(fmt.Sprint(r))
		}
		tsk := callstackPop()
		if tsk == nil {
			panic("no task on the callstack")
		}
		if recErr != nil {
			tsk.hasErrors = true
			tsk.isErrorsHandled = false
			tsk.err = recErr
		}

		tsk.errFunc(tsk)
		nextTask := callstackPeek()
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
		tsk.errFunc = nil
		tsk.err = nil
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
	callstackPush(tsk)
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
