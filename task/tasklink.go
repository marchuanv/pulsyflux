package task

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type callCxt int

const (
	DoCxt callCxt = iota
	RecvCxt
	ErrCtx
)

type callState int

const (
	Conclusive callState = iota
	Inconclusive
)

type task[T1 any, T2 any] struct {
	Id               string
	input            T1
	result           T2
	err              error
	errorHandled     bool
	errOnHandle      error
	doFunc           func(input T1) T2
	receiveFunc      func(results T2, input T1)
	errorFunc        func(err error, input T1) T1
	parent           *tskLink[T1, T2]
	syncChildren     map[string]*tskLink[T1, T2]
	asyncChildren    map[string]*tskLink[T1, T2]
	unlinkedChildren map[string]*tskLink[T1, T2]
	states           map[callCxt]callState
	isRoot           bool
	unlinkFlag       bool
}
type tskLink[T1 any, T2 any] task[T1, T2]

func newTskLink[T1 any, T2 any](input T1) *tskLink[T1, T2] {
	var result T2
	var err error
	var fatalErr error
	tLink := &tskLink[T1, T2]{
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
		map[string]*tskLink[T1, T2]{},
		map[string]*tskLink[T1, T2]{},
		map[string]*tskLink[T1, T2]{},
		map[callCxt]callState{},
		false,
		false,
	}
	return tLink
}

func (tLink *tskLink[T1, T2]) run() {
	tLink.callDoFunc()
	tLink.callReceiveFunc()
	tLink.callErrorFunc()
}

func (tLink *tskLink[T1, T2]) callDoFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tLink.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallDoFunc() error recovery: %s", tLink.err)
		}
	})()
	tLink.states[DoCxt] = Inconclusive
	tLink.result = tLink.doFunc(tLink.input)
	tLink.states[DoCxt] = Conclusive
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
		tLink.states[RecvCxt] = Inconclusive
		tLink.receiveFunc(tLink.result, tLink.input)
		tLink.states[RecvCxt] = Conclusive
	}
}

func (tLink *tskLink[T1, T2]) callErrorFunc() {
	if tLink.isRoot {
		if tLink.errorFunc == nil && tLink.is {

		} else {
			tLink.states[ErrCtx] = Inconclusive
			tLink.input = tLink.errorFunc(tLink.err, tLink.input)
			tLink.errorHandled = true
			tLink.states[ErrCtx] = Conclusive
		}
	} else {

		runErrFunc := false
		if parentTskLink == nil {
			runErrFunc = tLink.errorFunc != nil
		} else {
			state, ctxExists := parentTskLink.states[ErrCtx]
			runErrFunc = !ctxExists || (ctxExists && state == Inconclusive) && tLink.errorFunc != nil
		}
		if runErrFunc {
			tLink.states[ErrCtx] = Inconclusive
			tLink.input = tLink.errorFunc(tLink.err, tLink.input)
			tLink.errorHandled = true
			tLink.states[ErrCtx] = Conclusive
		}
	}
}

func (tLink *tskLink[T1, T2]) getUnlinkedLeafNode() *tskLink[T1, T2] {
	for _, child := range tLink.unlinkedChildren {
		return child.getUnlinkedLeafNode()
	}
	return tLink
}
