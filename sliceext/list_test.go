package sliceext

import (
	"testing"
)

type testType struct {
	id int
}

func TestList(test *testing.T) {

	test1 := testType{1}
	test2 := testType{2}
	test3 := testType{3}
	test4 := testType{4}

	list := NewList[testType]()

	list.Add(test1)
	list.Add(test2)
	list.Add(test3)
	list.Add(test4)

	list.Delete(test2)

	if !list.Has(test1) {
		test.Fail()
	}

	if list.Has(test2) {
		test.Fail()
	}

	if !list.Has(test3) {
		test.Fail()
	}

	if !list.Has(test3) {
		test.Fail()
	}

}
