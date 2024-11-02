package task

import (
	"errors"
	"testing"
)

func TestErrorHandle(test *testing.T) {
	outerMostRaised := false
	innerMostRaised := false
	Do(func() (string, error) {
		Do(func() (string, error) {
			Do(func() (string, error) {
				Do(func() (string, error) {
					return "", errors.New("something has gone wrong")
				}, func(err error, param string) string {
					innerMostRaised = true
					return param
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
		return param
	})
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
				}, nil)
				return "", nil
			}, nil)
			return "", nil
		}, nil)
		return "", nil
	}, nil)
}
