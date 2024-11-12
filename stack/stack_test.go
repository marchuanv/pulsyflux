package stack

import (
	"fmt"
	"testing"
)

func Test1Stack(test *testing.T) {
	stack := &Stack[string]{}
	for i := 0; i < 60; i++ {
		expected := fmt.Sprintf("Hello World %d", i)
		stack.Push(expected)
		cur, _ := stack.On(Push)
		if cur != expected {
			test.Fail()
		}
	}
}

func Test2Stack(test *testing.T) {
	stack := &Stack[string]{}
	for i := 0; i < 60; i++ {
		expected := fmt.Sprintf("Hello World %d", i)
		stack.Push(expected)
	}
	cur, _ := stack.On(Push)
	if cur != fmt.Sprintf("Hello World %d", 59) {
		test.Fail()
	}
}

func Test3Stack(test *testing.T) {
	stack := &Stack[string]{}
	stackExp := "Hello World 59"
	stackExp2 := "Hello World 0"
	stack.OnAsync(Push, func(current, previous string) {
		if current != stackExp2 {
			fmt.Printf("OnAsync: did not match expected, was: %s", current)
			test.Fail()
		}
	})
	for i := 0; i < 60; i++ {
		item := fmt.Sprintf("Hello World %d", i)
		stack.Push(item)
	}
	cur, _ := stack.On(Push)
	if cur != stackExp {
		fmt.Println("OnSync: did not match expected")
		test.Fail()
	}
}
