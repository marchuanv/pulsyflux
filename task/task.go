package task

import (
	"errors"
	"fmt"
	"pulsyflux/stack"

	"github.com/google/uuid"
)

type task struct {
	Id           uuid.UUID
	err          error
	errorParam   any
	errorHandled bool
	fatalErr     error
	result       any
	doFunc       func() any
	receiveFunc  func(variable any)
	errorFunc    func(err error, errorParam any) any
}

var taskStack = stack.Stack[*task]{}

func DoNow[T1 any, T2 any](doFunc func() T1, errorFuncs ...func(err error, errorParam T2) T2) T1 {
	var errorFunc func(err error, errorParam T2) T2
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	return execute(doFunc, nil, errorFunc)
}

func DoLater[T1 any, T2 any](doFunc func() T1, receive func(variable T1), errorFuncs ...func(err error, errorParam T2) T2) {
	var errorFunc func(err error, errorParam T2) T2
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	go execute(doFunc, receive, errorFunc)
}

func execute[T1 any, T2 any](doFunc func() T1, receive func(variable T1), errorFunc func(err error, errorParam T2) T2) T1 {
	defer (func() {
		currentTsk := taskStack.Pop()
		if currentTsk.fatalErr != nil {
			panic(currentTsk.err)
		}
		if currentTsk.err != nil {
			nextTsk := taskStack.Peek()
			if nextTsk == nil {
				if !currentTsk.errorHandled {
					panic(currentTsk.err)
				}
			} else {
				nextTsk.err = currentTsk.err
				nextTsk.errorParam = currentTsk.errorParam
				nextTsk.fatalErr = currentTsk.fatalErr
				nextTsk.errorHandled = currentTsk.errorHandled
			}
		}
	})()
	var result T1
	var err error
	var fatalErr error
	var errParam T2
	tsk := &task{
		uuid.New(),
		err,
		errParam,
		false,
		fatalErr,
		result,
		func() any {
			return doFunc()
		},
		func(variable any) {
			if receive != nil {
				receive(variable.(T1))
			}
		},
		func(err error, errorParam any) any {
			wantedErrorParam := errorParam.(T2)
			if errorFunc != nil {
				return errorFunc(err, wantedErrorParam)
			}
			return wantedErrorParam
		},
	}
	taskStack.Push(tsk)
	callDoFunc(tsk)
	callReceiveFunc(tsk)
	callErrorFunc(tsk)
	return tsk.result.(T1)
}

func callDoFunc(tsk *task) {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\nTask went into recovery. Error: %s", tsk.err)
		}
	})()
	if tsk.err == nil {
		tsk.result = tsk.doFunc()
	}
}

func callReceiveFunc(tsk *task) {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\nTask went into recovery. Error: %s", tsk.err)
		}
	})()
	if tsk.err == nil {
		tsk.receiveFunc(tsk.result)
	}
}

func callErrorFunc(tsk *task) {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.fatalErr = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\nTask went into recovery. Error: %s", tsk.fatalErr)
		}
	})()
	if tsk.err != nil {
		tsk.errorParam = tsk.errorFunc(tsk.err, tsk.errorParam)
		tsk.errorHandled = true
	}
}
