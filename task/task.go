package task

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Task[T1 any, T2 any] struct {
	Id           string
	input        T1
	result       T2
	err          error
	errorHandled bool
	fatalErr     error
	doFunc       func(input T1) T2
	receiveFunc  func(results T2, input T1)
	errorFunc    func(err error, input T1) T1
	parent       *TaskLink[T1, T2]
	children     map[string]*TaskLink[T1, T2]
	isAsync      bool
}

type TaskLink[T1 any, T2 any] Task[T1, T2]

type TaskCtx[T1 any, T2 any] struct {
	link      *TaskLink[T1, T2]
	callCount int
}

func NewTask[T1 any, T2 any](input T1) *TaskCtx[T1, T2] {
	var result T2
	var err error
	var fatalErr error
	tskLink := &TaskLink[T1, T2]{
		uuid.NewString(),
		input,
		result,
		err,
		false,
		fatalErr,
		nil,
		nil,
		nil,
		nil,
		nil,
		false,
	}
	tskLink.parent = nil
	tskLink.children = map[string]*TaskLink[T1, T2]{}
	return &TaskCtx[T1, T2]{tskLink, 0}
}

func (taskCtx *TaskCtx[T1, T2]) DoNow(doFunc func(input T1) T2, errorFuncs ...func(err error, input T1) T1) T2 {
	var errorFunc func(err error, input T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	taskCtx.link.doFunc = doFunc
	taskCtx.link.errorFunc = errorFunc
	execute(taskCtx, false)
	return taskCtx.link.result
}

func (taskCtx *TaskCtx[T1, T2]) DoLater(doFunc func(input T1) T2, receiveFunc func(results T2, input T1), errorFuncs ...func(err error, input T1) T1) {
	var errorFunc func(err error, errorParam T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	if receiveFunc == nil {
		panic("receive function is nil")
	}
	taskCtx.link.doFunc = doFunc
	taskCtx.link.receiveFunc = receiveFunc
	taskCtx.link.errorFunc = errorFunc
	go execute(taskCtx, true)
}

func execute[T1 any, T2 any](taskCtx *TaskCtx[T1, T2], isAsync bool) {
	defer (func() {
		err := recover()
		taskCtx.link.unlink()
		if err != nil {
			panic(err)
		}
	})()

	taskCtx.next(isAsync)
	newTsk := taskCtx.link
	newTsk.callDoFunc()
	newTsk.callReceiveFunc()
	newTsk.callErrorFunc()

	if newTsk.fatalErr != nil {
		panic(newTsk.fatalErr)
	}

	if newTsk.isAsync {
		if newTsk.err != nil {
			if !newTsk.errorHandled {
				panic(newTsk.err)
			}
		}
	} else {
		if newTsk.err != nil {
			if newTsk.parent == nil {
				if !newTsk.errorHandled {
					panic(newTsk.err)
				}
			} else {
				newTsk.parent.err = newTsk.err
				newTsk.parent.input = newTsk.input
				newTsk.parent.fatalErr = newTsk.fatalErr
				newTsk.parent.errorHandled = newTsk.errorHandled
			}
		}
	}
}

func (tsk *TaskLink[T1, T2]) callDoFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallDoFunc() error recovery: %s", tsk.err)
		}
	})()
	if tsk.err == nil {
		tsk.result = tsk.doFunc(tsk.input)
	}
}

func (tsk *TaskLink[T1, T2]) callReceiveFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallReceiveFunc() error recovery: %s", tsk.err)
		}
	})()
	if tsk.err == nil {
		if tsk.receiveFunc != nil {
			tsk.receiveFunc(tsk.result, tsk.input)
		}
	}
}

func (tsk *TaskLink[T1, T2]) callErrorFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tsk.fatalErr = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallErrorFunc() fatal error: %s", tsk.fatalErr)
		}
	})()
	if tsk.err != nil {
		if tsk.errorFunc != nil {
			tsk.input = tsk.errorFunc(tsk.err, tsk.input)
			tsk.errorHandled = true
		}
	}
}

func (taskLink *TaskLink[T1, T2]) new() {
	newTskLink := &TaskLink[T1, T2]{
		uuid.NewString(),
		taskLink.input,
		taskLink.result,
		taskLink.err,
		taskLink.errorHandled,
		taskLink.fatalErr,
		taskLink.doFunc,
		taskLink.receiveFunc,
		taskLink.errorFunc,
		taskLink,
		map[string]*TaskLink[T1, T2]{},
		taskLink.isAsync,
	}
	taskLink.children[newTskLink.Id] = newTskLink
}

func (node *TaskLink[T1, T2]) getLeafNode(isAsync bool) *TaskLink[T1, T2] {
	for _, child := range node.children {
		n := child.getLeafNode(isAsync)
		if n != nil {
			return n
		}
	}
	if node.isAsync && !isAsync {
		return nil
	}
	if !node.isAsync && isAsync {
		return nil
	}
	return node
}

func (nodeLink *TaskLink[T1, T2]) unlink() {
	if nodeLink.parent != nil {
		if len(nodeLink.children) > 0 {
			panic("fatal error node has children")
		}
		delete(nodeLink.parent.children, nodeLink.Id)
		nodeLink.parent = nil
	}
}

func (taskCtx *TaskCtx[T1, T2]) next(isAsync bool) {
	taskCtx.link = taskCtx.link.getLeafNode(isAsync)
	if taskCtx.callCount > 0 {
		taskCtx.link.new()
		taskCtx.link = taskCtx.link.getLeafNode(isAsync)
	}
	taskCtx.callCount += 1
}
