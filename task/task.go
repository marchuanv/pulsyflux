package task

import (
	"errors"
	"reflect"
)

type task struct {
	doFuncRef       reflect.Value
	errorFuncRef    reflect.Value
	errVal          reflect.Value
	errorParamVal   reflect.Value
	resultsVal      reflect.Value
	hasErrors       bool
	isErrorsHandled bool
	isChild         bool
}

var tskStack = taskStack{}
var taskCount = 0

func Do[T1 any, T2 any](doFunc func() (T1, error), errorFunc ...func(err error, errorParam T2) T2) T1 {
	var err error
	var errorParam T2
	var results T1
	doFuncRef := reflect.Indirect(reflect.ValueOf(doFunc))
	errVal := reflect.Indirect(reflect.ValueOf(err))
	errorParamVal := reflect.Indirect(reflect.ValueOf(errorParam))
	resultsVal := reflect.Indirect(reflect.ValueOf(results))
	var errorFuncRef reflect.Value
	if len(errorFunc) > 0 {
		errorFuncRef = reflect.Indirect(reflect.ValueOf(errorFunc[0]))
	}
	tsk := &task{
		doFuncRef,
		errorFuncRef,
		errVal,
		errorParamVal,
		resultsVal,
		false,
		false,
		false,
	}
	taskCount += 1
	if taskCount > 1 {
		tsk.isChild = true
	} else {
		tsk.isChild = false
	}
	callDoFunction(tsk)
	taskCount -= 1
	tskStack.Push(tsk)
	if tsk.isChild {
		tsk = nil
		return tskResult[T1](tsk)
	}
	for tskStack.Len() > 0 {
		popTsk := tskStack.Pop()
		if popTsk.hasErrors && !popTsk.errorFuncRef.IsZero() {
			isErrFuncAssign := tsk.errorFuncRef.Type().AssignableTo(popTsk.errorFuncRef.Type())
			isErrAssign := tsk.errVal.Type().AssignableTo(popTsk.errVal.Type())
			if isErrFuncAssign && isErrAssign {
				tsk.errVal = popTsk.errVal
				tsk.errorParamVal = popTsk.errorParamVal
				callErrorFunction(tsk)
				tsk.isErrorsHandled = true
			} else {
				unassignableErr := errors.New("one or more error function in the task chain does not have the correct function signatures")
				panic(unassignableErr)
			}
		}
	}
	if tsk.hasErrors && !tsk.isErrorsHandled {
		tskErr := tskError(tsk)
		panic(tskErr)
	}
	return tskResult[T1](tsk)
}

func callDoFunction(tsk *task) {
	out := tsk.doFuncRef.Call([]reflect.Value{})
	tsk.resultsVal = out[0]
	tsk.errVal = out[1]
	if !tsk.errVal.IsZero() {
		tsk.hasErrors = true
		tsk.isErrorsHandled = false
	}
	out = nil
}

func callErrorFunction(tsk *task) {
	out := tsk.errorFuncRef.Call([]reflect.Value{tsk.errVal, tsk.errorParamVal})
	tsk.errorParamVal = out[0]
	out = nil
}

func tskError(tsk *task) error {
	return tsk.errVal.Interface().(error)
}

func tskErrorParam[T any](tsk *task) T {
	return tsk.errorParamVal.Interface().(T)
}

func tskResult[T any](tsk *task) T {
	return tsk.resultsVal.Interface().(T)
}
