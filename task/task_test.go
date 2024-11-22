package task

import (
	"reflect"
	"testing"
)

func TestErrorHandle(test *testing.T) {
	expectedFuncCalls := []string{"func", "func1", "func11", "func111", "func1111", "func11111"}
	actualFuncCalls := []string{}
	expectedErrFuncCalls := []string{"errfunc", "errfunc1", "errfunc11", "errfunc111", "errfunc1111"}
	actualErrFuncCalls := []string{}
	input := "testdata"
	tsk := NewTskCtx[string, string](input)
	tsk.DoNow(func(in string) string {
		tsk.DoNow(func(in string) string {
			tsk.DoNow(func(in string) string {
				tsk.DoNow(func(in string) string {
					actualFuncCalls = append(actualFuncCalls, "func")
					panic("something went wrong")
				}, func(err error, in string) string {
					actualErrFuncCalls = append(actualErrFuncCalls, "errfunc")
					return "error occured here"
				})
				tsk.DoNow(func(in string) string {
					actualFuncCalls = append(actualFuncCalls, "func1")
					panic("something went wrong")
				}, func(err error, in string) string {
					actualErrFuncCalls = append(actualErrFuncCalls, "errfunc1")
					return "error occured here"
				})
				// //simulate a task that never returns
				// DoAsync(func() (string, error) {
				// 	actualFuncCalls = append(actualFuncCalls, "longrunningfunc")
				// 	return "hello", nil
				// }, func(val string) {

				// }, func(err error, param string) string {
				// 	actualErrFuncCalls = append(actualErrFuncCalls, "errfunc2")
				// 	return "error occured here"
				// })
				actualFuncCalls = append(actualFuncCalls, "func11")
				return ""
			}, func(err error, in string) string {
				actualErrFuncCalls = append(actualErrFuncCalls, "errfunc11")
				return in
			})
			actualFuncCalls = append(actualFuncCalls, "func111")
			return ""
		}, func(err error, in string) string {
			actualErrFuncCalls = append(actualErrFuncCalls, "errfunc111")
			return in
		})
		actualFuncCalls = append(actualFuncCalls, "func1111")
		return ""
	}, func(err error, in string) string {
		actualErrFuncCalls = append(actualErrFuncCalls, "errfunc1111")
		_param := tsk.DoNow(func(in string) string {
			actualFuncCalls = append(actualFuncCalls, "func11111")
			return in
		})
		return _param
	})
	if !reflect.DeepEqual(actualFuncCalls, expectedFuncCalls) {
		test.Log("function calls did not occur in the correct order")
		test.Fail()
	}
	if !reflect.DeepEqual(actualErrFuncCalls, expectedErrFuncCalls) {
		test.Log("error function calls did not occur in the correct order")
		test.Fail()
	}
}

func TestPanicNoErrorHandle(test *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			test.Errorf("The code did not panic")
		}
	}()
	input := "testdata"
	tsk := NewTskCtx[string, string](input)
	tsk.DoNow(func(in string) string {
		tsk.DoNow(func(in string) string {
			tsk.DoNow(func(in string) string {
				tsk.DoNow(func(in string) string {
					panic("something went wrong")
				})
				return ""
			})
			return ""
		})
		return ""
	})
}
