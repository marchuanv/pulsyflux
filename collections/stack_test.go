package collections

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestStack(test *testing.T) {
	someStruct := &struct{}{}
	stk := NewStack[*struct{}]()
	stk.Push(someStruct)
	if stk.Len() == 0 {
		test.Error("Expected stack length > 0 after push")
	}
	stk.Pop()
	if stk.Len() != 0 {
		test.Error("Expected stack length == 0 after pop")
	}
}

func TestStackConcurrency(test *testing.T) {
	msgA := "69b5430c-dd89-4a19-b5ff-7ddcf1c3fa0a"
	msgB := "d6daad92-0c0f-4b2e-9223-03a6dc3b5731"
	msgC := "690c0887-512f-468e-af48-734436467425"
	stk := NewStack[string]()
	var exit atomic.Bool
	go (func() {
		for !exit.Load() {
			stk.Push(msgA)
			pop := stk.Pop()
			if pop != msgA {
				test.Errorf("Expected %s, got %s", msgA, pop)
				exit.Store(true)
			}
			time.Sleep(3 * time.Second)
		}
	})()
	go (func() {
		for !exit.Load() {
			stk.Push(msgB)
			pop := stk.Pop()
			if pop != msgB {
				test.Errorf("Expected %s, got %s", msgB, pop)
				exit.Store(true)
			}
			time.Sleep(3 * time.Second)
		}
	})()
	go (func() {
		for !exit.Load() {
			stk.Push(msgC)
			pop := stk.Pop()
			if pop != msgC {
				test.Errorf("Expected %s, got %s", msgC, pop)
				exit.Store(true)
			}
			time.Sleep(3 * time.Second)
		}
	})()
	time.Sleep(15 * time.Second)
	exit.Store(true)
}
