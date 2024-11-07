package task

import (
	"errors"
	"testing"
)

func TestErrorHandle(test *testing.T) {
	outerMostRaised := false
	innerMostRaised := false
	innerMostExpectedParamValue := "inner parameter call"
	actualParamValue := "inner parameter call"
	Do(func() (string, error) {
		Do(func() (string, error) {
			Do(func() (string, error) {
				Do(func() (string, error) {
					return "", errors.New("something has gone wrong")
				}, func(err error, param string) string {
					innerMostRaised = true
					return "inner parameter call"
				})
				return "", nil
			}, func(err error, param string) string {
				return param
			})
			return "", nil
		}, func(err error, param string) string {
			return param
		})
		return "", nil
	}, func(err error, param string) string {
		outerMostRaised = true
		actualParamValue = param
		_param := Do[string, string](func() (string, error) {
			return param, nil
		})
		return _param
	})
	if innerMostExpectedParamValue != actualParamValue {
		test.Log("inner most parameter was not passed to outer most parameter")
		test.Fail()
	}
	if !outerMostRaised {
		test.Log("outer most error should have been raised")
		test.Fail()
	}
	if !innerMostRaised {
		test.Log("inner most error should have been raised")
		test.Fail()
	}
}

func TestPanicNoErrorHandle(test *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			test.Errorf("The code did not panic")
		}
	}()
	Do[string, any](func() (string, error) {
		Do[string, any](func() (string, error) {
			Do[string, any](func() (string, error) {
				Do[string, any](func() (string, error) {
					return "", errors.New("something has gone wrong")
				})
				return "", nil
			})
			return "", nil
		})
		return "", nil
	})
}
