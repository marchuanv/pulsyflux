package task

import (
	"errors"
	"testing"
)

func TestErrorHandle(test *testing.T) {
	outerMostRaised := false
	innerMostRaised := false
	Do(func() (string, string, error) {
		Do(func() (string, string, error) {
			Do(func() (string, string, error) {
				Do(func() (string, string, error) {
					return "", "", errors.New("something has gone wrong")
				}, func(err error, params ...string) {
					innerMostRaised = true
				})
				return "", "", nil
			}, func(err error, params ...string) {})
			return "", "", nil
		}, func(err error, params ...string) {})
		return "", "", nil
	}, func(err error, params ...string) {
		outerMostRaised = true
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
	Do(func() (string, string, error) {
		Do(func() (string, string, error) {
			Do(func() (string, string, error) {
				Do(func() (string, string, error) {
					return "", "", errors.New("something has gone wrong")
				})
				return "", "", nil
			})
			return "", "", nil
		})
		return "", "", nil
	})
}
