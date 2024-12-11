package sliceext

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStack(test *testing.T) {
	cloneStkId := uuid.New()
	someStruct := &struct{}{}

	stk := NewStack[*struct{}]()
	stkUnder := stk.(*stack[*struct{}])

	stk.Push(someStruct)
	stk.Clone(cloneStkId)

	if len(stkUnder.clones) == 0 {
		test.Fail()
	}

	clonePop := stk.ClonePop(cloneStkId)
	stkPop := stk.Pop()

	if len(stkUnder.clones) == 0 {
		test.Fail()
	}
	stk.CloneClear(cloneStkId)
	if len(stkUnder.clones) != 0 {
		test.Fail()
	}

	if clonePop != stkPop {
		test.Fail()
	}

	if stk.Len() != 0 || stk.CloneLen(cloneStkId) != 0 {
		test.Fail()
	}

}

func TestStackConcurrency(test *testing.T) {
	msgA := "69b5430c-dd89-4a19-b5ff-7ddcf1c3fa0a"
	msgB := "d6daad92-0c0f-4b2e-9223-03a6dc3b5731"
	msgC := "690c0887-512f-468e-af48-734436467425"
	stk := NewStack[string]()
	exit := false
	cloneStkId := uuid.New()
	go (func() {
		for !exit {
			stk.Push(msgA)
			stk.Push(msgB)
			stk.Push(msgC)
			time.Sleep(3 * time.Second)
		}
	})()
	go (func() {
		for !exit {
			stk.Clone(cloneStkId)
			msg := stk.ClonePop(cloneStkId)
			if msg != "" {
				test.Logf("Popped: %v\r\n", msg)
			}
			msg = stk.ClonePop(cloneStkId)
			if msg != "" {
				test.Logf("Popped: %v\r\n", msg)
			}
			msg = stk.ClonePop(cloneStkId)
			if msg != "" {
				test.Logf("Popped: %v\r\n", msg)
			}
			time.Sleep(100 * time.Millisecond)
		}
	})()
	time.Sleep(15 * time.Second)
	exit = true
}
