package task

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
)

type funcCall int

const (
	None funcCall = iota
	DoFunc
	ReceiveFunc
	ErrorFunc
)

type task[T1 any, T2 any] struct {
	Id           string
	input        T1
	result       T2
	err          error
	errorHandled bool
	doFunc       func(input T1) T2
	receiveFunc  func(results T2, input T1)
	errorFunc    func(err error, input T1) T1
	parent       *tskLink[T1, T2]
	children     []*tskLink[T1, T2]
	isRoot       bool
	lstFuncCall  funcCall
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
		[]*tskLink[T1, T2]{},
		false,
		None,
	}
	return tLink
}

func (tLink *tskLink[T1, T2]) run() {
	if tLink.err == nil {
		tLink.callDoFunc()
		tLink.callReceiveFunc()
	}
	tLink.callErrorFunc()
	if tLink.isRoot {
		if tLink.err != nil && !tLink.errorHandled {
			panic(tLink.err)
		}
	} else if tLink.err != nil {
		tLink.parent.err = tLink.err
		tLink.parent.errorHandled = tLink.errorHandled
	}
}

func (tLink *tskLink[T1, T2]) callDoFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tLink.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallDoFunc() error recovery: %s", tLink.err)
		}
	})()
	tLink.lstFuncCall = DoFunc
	tLink.result = tLink.doFunc(tLink.input)
	tLink.lstFuncCall = None
}

func (tLink *tskLink[T1, T2]) callReceiveFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tLink.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallReceiveFunc() error recovery: %s", tLink.err)
		}
	})()
	if tLink.receiveFunc != nil {
		tLink.lstFuncCall = ReceiveFunc
		tLink.receiveFunc(tLink.result, tLink.input)
		tLink.lstFuncCall = None
	}
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
	if tLink.errorFunc != nil {
		tLink.lstFuncCall = ErrorFunc
		tLink.input = tLink.errorFunc(tLink.err, tLink.input)
		tLink.errorHandled = true
		tLink.lstFuncCall = None
	}
}

func (tLink *tskLink[T1, T2]) getLeafNode() *tskLink[T1, T2] {
	for _, child := range tLink.children {
		return child.getLeafNode()
	}
	return tLink
}

func (tLink *tskLink[T1, T2]) getNodeBy(lastFuncCall funcCall) *tskLink[T1, T2] {
	for _, child := range tLink.children {
		c := child.getNodeBy(lastFuncCall)
		if c != nil {
			return c
		}
	}
	if tLink.lstFuncCall == lastFuncCall {
		return tLink
	}
	return nil
}

// func children(*tskLink[T1, T2]) {
// 	for _, child := range tLink.children {
// 		c := child.getNodeBy(lastFuncCall)
// 		if c != nil {
// 			return c
// 		}
// 	}
// 	if tLink.lstFuncCall == lastFuncCall {
// 		return tLink
// 	}
// 	return nil
// }

// func (tLink *tskLink[T1, T2]) unlink() {
// 	if len(tLink.children) > 0 {
// 		panic("fatal error node has children")
// 	}
// 	if tLink.parent != nil {
// 		delete(tLink.parent.children, tLink.Id)
// 	}
// }

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
		[]*tskLink[T1, T2]{},
		false,
		None,
	}
	tLink.children = append(tLink.children, newTskLink)
	slices.Reverse(tLink.children)
}

func (tLink *tskLink[T1, T2]) newChildTsk(input T1) {
	newTskLink := newTskLink[T1, T2](input)
	newTskLink.parent = tLink
	tLink.children = append(tLink.children, newTskLink)
	slices.Reverse(tLink.children)
}
