package task

import (
	"errors"
	"reflect"
)

type task struct {
	doFuncRef     reflect.Value
	errorFuncRef  reflect.Value
	errVal        reflect.Value
	errorParamVal reflect.Value
	resultsVal    reflect.Value
}

var errorStack = taskStack{}
var emptyParameters = []reflect.Value{}

func Do[T any, T2 any](doFunc func() (T, error), errorFunc ...func(err error, errorParam T2) T2) T {
	var err error
	var errorParam T2
	var results T
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
	}

	if len(errorStack) == 0 {
		values := tsk.doFuncRef.Call(emptyParameters)
		tsk.resultsVal = reflect.Indirect(reflect.ValueOf(values[0]))
		tsk.errVal = reflect.Indirect(reflect.ValueOf(values[1]))
		if tsk.errVal.IsZero() {
			results = tsk.resultsVal.Interface().(T)
		} else {
			err = tsk.resultsVal.Interface().(error)
			if !tsk.errorFuncRef.IsZero() {
				callErrorFunction(tsk)
			}
			errorStack.Push(tsk)
		}
	} else {
		var tskErr *task
		errorStack, tskErr = errorStack.Pop()
		if !tsk.errorFuncRef.IsZero() {
			isErrFuncAssign := tsk.errorFuncRef.Type().AssignableTo(tskErr.errorFuncRef.Type())
			isErrAssign := tsk.errVal.Type().AssignableTo(tskErr.errVal.Type())
			if isErrFuncAssign && isErrAssign {
				tsk.errVal = tskErr.errVal
				tsk.errorParamVal = tskErr.errorParamVal
				callErrorFunction(tsk)
			} else {
				err = errors.New("one or more error function in the task chain does not have the correct function signatures")
				panic(err)
			}
		}
		errorStack.Push(tskErr)
	}
	return results
}
func callErrorFunction(tsk *task) {
	values := []reflect.Value{tsk.errVal, tsk.errorParamVal}
	values = tsk.errorFuncRef.Call(values)
	tsk.errorParamVal = reflect.Indirect(reflect.ValueOf(values[0]))
	values = nil
}
