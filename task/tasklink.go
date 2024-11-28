package task

import (
	"errors"
	"fmt"
	"pulsyflux/stack"

	"github.com/google/uuid"
)

type funcCall string

const (
	DoFunc      funcCall = "pulsyflux/task.(*tskLink[...]).callDoFunc"
	ReceiveFunc funcCall = "pulsyflux/task.(*tskLink[...]).callReceiveFunc"
	ErrorFunc   funcCall = "pulsyflux/task.(*tskLink[...]).callErrorFunc"
)

type task[T1 any, T2 any] struct {
	Id            string
	input         T1
	result        T2
	err           error
	errorHandled  bool
	doFunc        func(input T1) T2
	receiveFunc   func(results T2, input T1)
	errorFunc     func(err error, input T1) T1
	parent        *tskLink[T1, T2]
	children      *stack.Stack[*tskLink[T1, T2]]
	isRoot        bool
	funcCallstack *stack.Stack[funcCall]
}
type tskLink[T1 any, T2 any] task[T1, T2]

func newTskLink[T1 any, T2 any](input T1) *tskLink[T1, T2] {
	var result T2
	var err error
	tLink := &tskLink[T1, T2]{
		uuid.NewString(),
		input,
		result,
		err,
		false,
		nil,
		nil,
		nil,
		nil,
		stack.NewStack[*tskLink[T1, T2]](),
		false,
		stack.NewStack[funcCall](),
	}
	return tLink
}

func (tLink *tskLink[T1, T2]) run() {
	if tLink.err == nil {
		tLink.callDoFunc()
		if tLink.receiveFunc != nil {
			tLink.callReceiveFunc()
		}
	}
	if tLink.err != nil && tLink.errorFunc != nil {
		tLink.callErrorFunc()
	}
	if tLink.isRoot {
		if tLink.err != nil && !tLink.errorHandled {
			panic(tLink.err)
		}
	} else if tLink.err != nil {
		tLink.parent.err = tLink.err
		tLink.parent.errorHandled = tLink.errorHandled
	}
	tLink.unlink()
}

func (tLink *tskLink[T1, T2]) callDoFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tLink.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallDoFunc() error recovery: %s", tLink.err)
		}
	})()
	tLink.updateCallstack()
	tLink.result = tLink.doFunc(tLink.input)
}

func (tLink *tskLink[T1, T2]) callReceiveFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tLink.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallReceiveFunc() error recovery: %s", tLink.err)
		}
	})()
	tLink.updateCallstack()
	tLink.receiveFunc(tLink.result, tLink.input)
}

func (tLink *tskLink[T1, T2]) callErrorFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tLink.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallErrorFunc() error recovery: %s", tLink.err)
			tLink.errorHandled = false
		}
	})()
	tLink.updateCallstack()
	tLink.input = tLink.errorFunc(tLink.err, tLink.input)
	tLink.errorHandled = true
}

func (tLink *tskLink[T1, T2]) getLeafNode(filters ...funcCall) *tskLink[T1, T2] {
	child := tLink.children.Peek()
	if child != nil {
		return child.getLeafNode(filters...)
	} else {
		for _, filter := range filters {
			if tLink.funcCallstack.Peek() == filter {
				return tLink
			}
		}
		return tLink
	}
}

func (tLink *tskLink[T1, T2]) findNode(fCall funcCall) *tskLink[T1, T2] {
	var found *tskLink[T1, T2]
	if tLink.funcCallstack.Peek() == fCall {
		found = tLink
	}
	if found == nil {
		child := tLink.children.Peek()
		if child != nil {
			return child.findNode(fCall)
		}
	}
	return found
}

func (tLink *tskLink[T1, T2]) updateCallstack() {
	clstk := getCallstack()
	for clstk.Len() > 0 {
		funcName := clstk.Pop()
		switch funcName {
		case string(DoFunc):
			tLink.funcCallstack.Push(DoFunc)
		case string(ReceiveFunc):
			tLink.funcCallstack.Push(ReceiveFunc)
		case string(ErrorFunc):
			tLink.funcCallstack.Push(ErrorFunc)
		}
	}
}

func (tLink *tskLink[T1, T2]) unlink() {
	if tLink.children.Len() > 0 {
		panic("fatal error node has children")
	}
	if tLink.parent != nil {
		tLink.parent.children.Pop()
	}
}

func (tLink *tskLink[T1, T2]) newChildClnTsk() {
	newTskLink := &tskLink[T1, T2]{
		uuid.NewString(),
		tLink.input,
		tLink.result,
		tLink.err,
		tLink.errorHandled,
		tLink.doFunc,
		tLink.receiveFunc,
		tLink.errorFunc,
		tLink,
		stack.NewStack[*tskLink[T1, T2]](),
		false,
		stack.NewStack[funcCall](),
	}
	tLink.children.Push(newTskLink)
}

func (tLink *tskLink[T1, T2]) newChildTsk(input T1) {
	newTskLink := newTskLink[T1, T2](input)
	newTskLink.parent = tLink
	tLink.children.Push(newTskLink)
}
