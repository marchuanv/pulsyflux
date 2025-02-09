package sliceext

import (
	"testing"
	"time"
)

func TestStack(test *testing.T) {
	someStruct := &struct{}{}
	stk := NewStack[*struct{}]()
	stk.Push(someStruct)
	if stk.Len() == 0 {
		test.Fail()
	}
	stk.Pop()
	if stk.Len() != 0 {
		test.Fail()
	}
}

func TestStackConcurrency(test *testing.T) {
	msgA := "69b5430c-dd89-4a19-b5ff-7ddcf1c3fa0a"
	msgB := "d6daad92-0c0f-4b2e-9223-03a6dc3b5731"
	msgC := "690c0887-512f-468e-af48-734436467425"
	stk := NewStack[string]()
	exit := false
	go (func() {
		for !exit {
			stk.Push(msgA)
			pop := stk.Pop()
			if pop != msgA {
				test.Fail()
				exit = true
			}
			time.Sleep(3 * time.Second)
		}
	})()
	go (func() {
		for !exit {
			stk.Push(msgB)
			pop := stk.Pop()
			if pop != msgB {
				test.Fail()
				exit = true
			}
			time.Sleep(3 * time.Second)
		}
	})()
	go (func() {
		for !exit {
			stk.Push(msgC)
			pop := stk.Pop()
			if pop != msgC {
				test.Fail()
				exit = true
			}
			time.Sleep(3 * time.Second)
		}
	})()
	time.Sleep(15 * time.Second)
	exit = true
}
