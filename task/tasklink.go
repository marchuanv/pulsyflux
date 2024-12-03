package task

import (
	"errors"
	"fmt"
	"pulsyflux/sliceext"

	"github.com/google/uuid"
)

type funcCall string

const (
	DoFunc    funcCall = "pulsyflux/task.(*tskLink).callDoFunc"
	ErrorFunc funcCall = "pulsyflux/task.(*tskLink).callErrorFunc"
)

type task struct {
	Id            string
	channel       *chnl
	err           error
	errorHandled  bool
	doFunc        func(channel Channel)
	errorFunc     func(err error, channel Channel)
	parent        *tskLink
	children      *sliceext.Stack[*tskLink]
	isRoot        bool
	funcCallstack *sliceext.Stack[funcCall]
}
type tskLink task

func newTskLink() *tskLink {
	var err error
	channel := newChl()
	return &tskLink{
		uuid.NewString(),
		channel,
		err,
		false,
		nil,
		nil,
		nil,
		sliceext.NewStack[*tskLink](),
		false,
		sliceext.NewStack[funcCall](),
	}
}

func (tLink *tskLink) newChildTskClone() {
	newTskLink := tLink.newChildTsk()
	newTskLink.err = tLink.err
	newTskLink.errorHandled = tLink.errorHandled
}

func (tLink *tskLink) newChildTsk() *tskLink {
	newTskLink := newTskLink()
	newTskLink.parent = tLink
	tLink.children.Push(newTskLink)
	return newTskLink
}

func (tLink *tskLink) run() {
	go (func() {
		if tLink.err == nil {
			tLink.callDoFunc()
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
	})()
	tLink.channel.Read(func(data any) {
		fmt.Printf("\r\n%s\r\n", data)
	})
	tLink.channel.close()
}

func (tLink *tskLink) callDoFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tLink.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallDoFunc() error recovery: %s", tLink.err)
		}
	})()
	tLink.updateCallstack()
	tLink.doFunc(tLink.channel)
}

func (tLink *tskLink) callErrorFunc() {
	defer (func() {
		r := recover()
		if r != nil {
			tLink.err = errors.New(fmt.Sprint(r))
			fmt.Printf("\r\ncallErrorFunc() error recovery: %s", tLink.err)
			tLink.errorHandled = false
		}
	})()
	tLink.updateCallstack()
	tLink.errorFunc(tLink.err, tLink.channel)
	tLink.errorHandled = true
}

func (tLink *tskLink) getLeafNode(filters ...funcCall) *tskLink {
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

func (tLink *tskLink) updateCallstack() {
	clstk := getCallstack()
	for clstk.Len() > 0 {
		funcName := clstk.Pop()
		switch funcName {
		case string(DoFunc):
			tLink.funcCallstack.Push(DoFunc)
		case string(ErrorFunc):
			tLink.funcCallstack.Push(ErrorFunc)
		}
	}
}

func (tLink *tskLink) unlink() {
	if tLink.children.Len() > 0 {
		panic("fatal error node has children")
	}
	if tLink.parent != nil {
		tLink.parent.children.Pop()
	}
}
