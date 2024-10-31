package task

import (
	"errors"
	"testing"
)

func TestOuterMostErrorHandle(test *testing.T) {
	outerMostRaised := false
	Do[string, string](func() (string, error) {
		results, _ := Do[string, string](func() (string, error) {
			results, _ := Do[string, string](func() (string, error) {
				results, _ := Do[string, string](func() (string, error) {
					return "", errors.New("something has gone wrong")
				}, func(err error, params ...string) string {
					test.Log("error should not have been raised at this level")
					test.Fail()
					return ""
				})
				return results, nil
			}, func(err error, params ...string) string {
				test.Log("error should not have been raised at this level")
				test.Fail()
				return ""
			})
			return results, nil
		}, func(err error, params ...string) string {
			test.Log("error should not have been raised at this level")
			test.Fail()
			return ""
		})
		return results, nil
	}, func(err error, params ...string) string {
		outerMostRaised = true
		return ""
	})
	if !outerMostRaised {
		test.Log("error should have been raised")
		test.Fail()
	}
}

func TestInnerMostErrorHandle(test *testing.T) {
	innerMostRaised := false
	hasParameters := false
	Do[string, string](func() (string, error) {
		results, _ := Do[string, string](func() (string, error) {
			results, _ := Do[string, string](func() (string, error) {
				results, _ := Do[string, string](func() (string, error) {
					return "some params passed to error func", errors.New("something has gone wrong")
				}, func(err error, params ...string) string {
					if params[0] == "some params passed to error func" {
						hasParameters = true
					}
					innerMostRaised = true
					return ""
				})
				return results, nil
			})
			return results, nil
		})
		return results, nil
	})
	if !innerMostRaised {
		test.Log("error should have been raised")
		test.Fail()
	}
	if !hasParameters {
		test.Log("expected parameters passed to the error fail function")
		test.Fail()
	}
}

func TestPanicNoErrorHandle(test *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			test.Errorf("The code did not panic")
		}
	}()
	Do[string, string](func() (string, error) {
		results, _ := Do[string, string](func() (string, error) {
			results, _ := Do[string, string](func() (string, error) {
				results, _ := Do[string, string](func() (string, error) {
					return "", errors.New("something has gone wrong")
				})
				return results, nil
			})
			return results, nil
		})
		return results, nil
	})
}

// type test[T any] struct {
// }

// type test2[T string] struct {
// }

// func TestSliceGenerics() {

// 	arr := make([]interface{}, 10)

// 	t1 := test[int]{}
// 	arr = append(arr, t1)

// }
