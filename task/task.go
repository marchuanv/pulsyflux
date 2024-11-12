package task

import (
	"errors"
	"fmt"
	"pulsyflux/stack"
	"reflect"
)

type task struct {
	results         reflect.Value
	err             error
	errorParam      reflect.Value
	hasErrors       bool
	isErrorsHandled bool
	isExecuted      bool
	doFunc          reflect.Value
	receiveFunc     reflect.Value
	errFunc         reflect.Value
}

var tskExecuteStack = stack.Stack[*task]{}
var tskExecutingStack = stack.Stack[*task]{}
var tskErrorStack = stack.Stack[*task]{}
var tskDoneStack = stack.Stack[*task]{}

func DoNow[T1 any, T2 any](doFunc func() (T1, error), errorFuncs ...func(err error, errorParam T2) T2) T1 {
	tsk := createTask()
	var errorFunc = errorFuncs[0]
	return execute(doFunc, nil, errorFunc, tsk)
}

func DoLater[T1 any, T2 any](doFunc func() (T1, error), receive func(variable T1), errorFuncs ...func(err error, errorParam T2) T2) {
	tsk := createTask()
	var errorFunc = errorFuncs[0]
	go execute(doFunc, receive, errorFunc, tsk)
}

func execute[T1 any, T2 any](doFunc func() (T1, error), receive func(variable T1), errorFunc func(err error, errorParam T2) T2, tsk *task) T1 {
	tskExecuteStack.OnAsync(stack.Push, func(newTask *task, currentTask *task) {
		callDoFunc[T1](newTask)
		if currentTask != nil && currentTask.hasErrors {
			newTask.hasErrors = true
			newTask.isErrorsHandled = currentTask.isErrorsHandled
			newTask.err = currentTask.err
			if !isZero(currentTask.errorParam) {
				newTask.errorParam = currentTask.errorParam
			}
		}
		tskExecutingStack.Push(newTask)
	})
	tskExecutingStack.OnAsync(stack.Push, func(newTask *task, currentTask *task) {
		if newTask.hasErrors {
			if !isZero(newTask.errFunc) {
				tskErrorStack.Push(newTask)
			}
		} else {
			if !isZero(newTask.receiveFunc) {
				callReceiveFunc[T1](newTask)
			}
			if newTask.hasErrors {
				if !isZero(newTask.errFunc) {
					tskErrorStack.Push(newTask)
				}
			} else {
				tskDoneStack.Push(newTask)
			}
		}
		tskExecutingStack.Pop()
	})
	tskErrorStack.OnAsync(stack.Push, func(newTask *task, currentTask *task) {
		callErrorFunc[T2](newTask)
		tskErrorStack.Pop()
		tskDoneStack.Push(newTask)
	})
	tsk.doFunc = reflect.Indirect(reflect.ValueOf(doFunc))
	if receive != nil {
		tsk.receiveFunc = reflect.Indirect(reflect.ValueOf(receive))
	}
	if errorFunc != nil {
		tsk.errFunc = reflect.Indirect(reflect.ValueOf(errorFunc))
	}
	tskExecuteStack.Push(tsk)
	newTask, _ := tskDoneStack.On(stack.Push)
	results := newTask.results.Interface().(T1)
	tskDoneStack.Pop()
	return results
}

func callReceiveFunc[T any](tsk *task) {
	defer (func() {
		var recErr error
		r := recover()
		if r != nil {
			recErr = errors.New(fmt.Sprint(r))
			fmt.Printf("callReceiveFunc went into recovery. Error: %s", recErr)
			tsk.hasErrors = true
			tsk.isErrorsHandled = false
			tsk.err = recErr
		}
	})()
	receive := tsk.receiveFunc.Interface().(func(variable T))
	results := tsk.results.Interface().(T)
	receive(results)
}

func callDoFunc[T any](tsk *task) {
	defer (func() {
		var recErr error
		r := recover()
		if r != nil {
			recErr = errors.New(fmt.Sprint(r))
			fmt.Printf("callDoFunc went into recovery. Error: %s", recErr)
			tsk.hasErrors = true
			tsk.isErrorsHandled = false
			tsk.err = recErr
		}
	})()
	doFunc := tsk.doFunc.Interface().(func() (T, error))
	results, tskErr := doFunc()
	tsk.results = reflect.ValueOf(results)
	tsk.isExecuted = true
	tsk.isErrorsHandled = false
	tsk.hasErrors = false
	if tskErr != nil {
		tsk.hasErrors = true
		tsk.err = tskErr
	}
}

func callErrorFunc[T any](tsk *task) {
	defer (func() {
		var recErr error
		r := recover()
		if r != nil {
			recErr = errors.New(fmt.Sprint(r))
			fmt.Printf("callErrorFunc went into recovery. Error: %s", recErr)
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
	errorFunc := tsk.errFunc.Interface().(func(err error, errorParam T) T)
	errParam = errorFunc(tsk.err, errParam)
	tsk.errorParam = reflect.ValueOf(errParam)
	tsk.isErrorsHandled = true
}

func createTask() *task {
	return &task{
		reflect.Value{},
		nil,
		reflect.Value{},
		false,
		true,
		false,
		reflect.Value{},
		reflect.Value{},
		reflect.Value{},
	}
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
