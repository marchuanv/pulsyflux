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
	errorHandled bool
	fatalErr     error
	result       any
	doFunc       func(t *task)
	receiveFunc  func(t *task)
	errorFunc    func(t *task)
	input        any
}

var taskStack = stack.Stack[*task]{}

func DoNow[T1 any, T2 any](input T1, doFunc func(input T1) T2, errorFuncs ...func(err error, input T1) T1) T2 {
	var errorFunc func(err error, input T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	return execute(input, doFunc, nil, errorFunc)
}

func DoLater[T1 any, T2 any](input T1, doFunc func(input T1) T2, receive func(results T2, input T1), errorFuncs ...func(err error, input T1) T1) {
	var errorFunc func(err error, errorParam T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	go execute(input, doFunc, receive, errorFunc)
}

func execute[T1 any, T2 any](input T1, doFunc func(input T1) T2, receive func(results T2, input T1), errorFunc func(err error, input T1) T1) T2 {
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
				nextTsk.input = currentTsk.input
				nextTsk.fatalErr = currentTsk.fatalErr
				nextTsk.errorHandled = currentTsk.errorHandled
			}
		}
	})()
	var result T2
	var err error
	var fatalErr error
	tsk := &task{
		uuid.New(),
		err,
		false,
		fatalErr,
		result,
		func(t *task) {
			if t.err == nil {
				t.result = doFunc(t.input.(T1))
			}
		},
		func(t *task) {
			if t.err == nil {
				if receive != nil {
					receive(t.result.(T2), t.input.(T1))
				}
			}
		},
		func(t *task) {
			if t.err != nil {
				if errorFunc != nil {
					t.errorHandled = true
					t.input = errorFunc(t.err, t.input.(T1))
				}
			}
		},
		input,
	}
	taskStack.Push(tsk)
	callDoFunc(tsk)
	callReceiveFunc(tsk)
	callErrorFunc(tsk)
	return tsk.result.(T2)
}

func callDoFunc(tsk *task) {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\nTask went into recovery. Error: %s", tsk.err)
		}
	})()
	tsk.doFunc(tsk)
}

func callReceiveFunc(tsk *task) {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\nTask went into recovery. Error: %s", tsk.err)
		}
	})()
	tsk.receiveFunc(tsk)
}

func callErrorFunc(tsk *task) {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.fatalErr = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\nTask went into recovery. Error: %s", tsk.fatalErr)
		}
	})()
	tsk.errorFunc(tsk)
}
