package task

import (
	"errors"
	"reflect"
	"testing"
)

func TestErrorHandle(test *testing.T) {
	expectedFuncCalls := []string{"func", "func1", "func11", "func111", "func1111"}
	actualFuncCalls := []string{}
	expectedErrFuncCalls := []string{"errfunc", "errfunc1", "errfunc11", "errfunc111", "errfunc1111"}
	actualErrFuncCalls := []string{}
	DoNow(func() (string, error) {
		DoNow(func() (string, error) {
			DoNow(func() (string, error) {
				DoNow(func() (string, error) {
					actualFuncCalls = append(actualFuncCalls, "func")
					return "", errors.New("something has gone wrong")
				}, func(err error, param string) string {
					actualErrFuncCalls = append(actualErrFuncCalls, "errfunc")
					return "error occured here"
				})
				DoNow(func() (string, error) {
					actualFuncCalls = append(actualFuncCalls, "func1")
					return "", errors.New("something has gone wrong")
				}, func(err error, param string) string {
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
				return "", nil
			}, func(err error, param string) string {
				actualErrFuncCalls = append(actualErrFuncCalls, "errfunc11")
				return param
			})
			actualFuncCalls = append(actualFuncCalls, "func111")
			return "", nil
		}, func(err error, param string) string {
			actualErrFuncCalls = append(actualErrFuncCalls, "errfunc111")
			return param
		})
		actualFuncCalls = append(actualFuncCalls, "func1111")
		return "", nil
	}, func(err error, param string) string {
		actualErrFuncCalls = append(actualErrFuncCalls, "errfunc1111")
		_param := DoNow[string, string](func() (string, error) {
			return param, nil
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
	DoNow[string, any](func() (string, error) {
		DoNow[string, any](func() (string, error) {
			DoNow[string, any](func() (string, error) {
				DoNow[string, any](func() (string, error) {
					return "", errors.New("something has gone wrong")
				})
				return "", nil
			})
			return "", nil
		})
		return "", nil
	})
}
