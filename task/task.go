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
	isAsync      bool
	err          error
	errorHandled bool
	fatalErr     error
	doFunc       func(input T1) T2
	receiveFunc  func(results T2, input T1)
	errorFunc    func(err error, input T1) T1
	rootLink     *NodeLink[T1, T2]
}

type TaskNode[T1 any, T2 any] struct {
	task     *Task[T1, T2]
	parent   *NodeLink[T1, T2]
	children map[string]*NodeLink[T1, T2]
}

type NodeLink[T1 any, T2 any] TaskNode[T1, T2]

func NewTask[T1 any, T2 any](input T1) *Task[T1, T2] {
	var result T2
	var err error
	var fatalErr error
	return &Task[T1, T2]{
		uuid.NewString(),
		input,
		result,
		false,
		err,
		false,
		fatalErr,
		nil,
		nil,
		nil,
		nil,
	}
}

func (tsk *Task[T1, T2]) DoNow(doFunc func(input T1) T2, errorFuncs ...func(err error, input T1) T1) T2 {
	var errorFunc func(err error, input T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	tsk.doFunc = doFunc
	tsk.errorFunc = errorFunc
	tsk.isAsync = false
	execute[T1, T2](tsk)
	return tsk.result
}

func (tsk *Task[T1, T2]) DoLater(doFunc func(input T1) T2, receiveFunc func(results T2, input T1), errorFuncs ...func(err error, input T1) T1) {
	var errorFunc func(err error, errorParam T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	if receiveFunc == nil {
		panic("receive function is nil")
	}
	tsk.doFunc = doFunc
	tsk.receiveFunc = receiveFunc
	tsk.errorFunc = errorFunc
	tsk.isAsync = true
	go execute(tsk)
}

func execute[T1 any, T2 any](tsk *Task[T1, T2]) {

	if tsk.rootLink == nil {
		tsk.rootLink = &NodeLink[T1, T2]{tsk, nil, map[string]*NodeLink[T1, T2]{}}
	}

	parentLink := tsk.rootLink.getLeafNode(tsk.isAsync)
	newTsk := tsk.new()
	parentLink.newLink(newTsk)
	tskLink := tsk.rootLink.getLeafNode(tsk.isAsync)

	newTsk.callDoFunc()
	newTsk.callReceiveFunc()
	newTsk.callErrorFunc()

	if !newTsk.isAsync {
		if newTsk.fatalErr != nil {
			panic(newTsk.fatalErr)
		}
		if newTsk.err != nil {
			if tskLink.parent == nil {
				if !newTsk.errorHandled {
					panic(newTsk.err)
				}
			} else {
				tskLink.parent.task.err = newTsk.err
				tskLink.parent.task.input = newTsk.input
				tskLink.parent.task.fatalErr = newTsk.fatalErr
				tskLink.parent.task.errorHandled = newTsk.errorHandled
			}
		} else {
			if newTsk.fatalErr != nil {
				panic(newTsk.err)
			}
			if newTsk.err != nil {
				if !newTsk.errorHandled {
					parentLink.unlink(newTsk)
					panic(newTsk.err)
				}
			}
		}
	}
	parentLink.unlink(newTsk)
}

func (tsk *Task[T1, T2]) callDoFunc() {
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

func (tsk *Task[T1, T2]) callReceiveFunc() {
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

func (tsk *Task[T1, T2]) callErrorFunc() {
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

func (tsk *Task[T1, T2]) new() *Task[T1, T2] {
	return &Task[T1, T2]{
		uuid.NewString(),
		tsk.input,
		tsk.result,
		tsk.isAsync,
		tsk.err,
		tsk.errorHandled,
		tsk.fatalErr,
		tsk.doFunc,
		tsk.receiveFunc,
		tsk.errorFunc,
		nil,
	}
}

func (nodeLink *NodeLink[T1, T2]) newLink(tsk *Task[T1, T2]) {
	newLink := &NodeLink[T1, T2]{
		tsk,
		nodeLink,
		map[string]*NodeLink[T1, T2]{},
	}
	nodeLink.children[tsk.Id] = newLink
}

func (node *NodeLink[T1, T2]) getLeafNode(isAsync bool) *NodeLink[T1, T2] {
	for _, child := range node.children {
		n := child.getLeafNode(isAsync)
		if n != nil {
			return n
		}
	}
	if node.task.isAsync && !isAsync {
		return nil
	}
	if !node.task.isAsync && isAsync {
		return nil
	}
	return node
}

func (nodeLink *NodeLink[T1, T2]) unlink(tsk *Task[T1, T2]) {
	child, exists := nodeLink.children[tsk.Id]
	if exists {
		delete(nodeLink.children, tsk.Id)
		child.parent = nil
	} else {
		panic("fatal error task is not a child")
	}
}
