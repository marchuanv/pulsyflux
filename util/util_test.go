package util

import (
	"errors"
	"testing"
)

func TestIsValidUUID(test *testing.T) {
	isValid := IsValidUUID("831d2c15-1a90-413e-9189-4cff18b5ae9c")
	if !isValid {
		test.Fail()
	}
}
func TestIsNotValidUUID(test *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			test.Errorf("The code did not panic")
		}
	}()
	IsValidUUID("awdawdawdwad")
}

func TestOuterMostErrorHandle(test *testing.T) {
	outerMostRaised := false
	Do(func() (string, error) {
		results := Do(func() (string, error) {
			results := Do(func() (string, error) {
				results := Do(func() (string, error) {
					return "", errors.New("something has gone wrong")
				}, func(err error) {
					test.Log("error should not have been raised at this level")
					test.Fail()
				})
				return results, nil
			}, func(err error) {
				test.Log("error should not have been raised at this level")
				test.Fail()
			})
			return results, nil
		}, func(err error) {
			test.Log("error should not have been raised at this level")
			test.Fail()
		})
		return results, nil
	}, func(err error) {
		outerMostRaised = true
	})
	if !outerMostRaised {
		test.Log("error should have been raised")
		test.Fail()
	}
}

func TestInnerMostErrorHandle(test *testing.T) {
	innerMostRaised := false
	Do(func() (string, error) {
		results := Do(func() (string, error) {
			results := Do(func() (string, error) {
				results := Do(func() (string, error) {
					return "", errors.New("something has gone wrong")
				}, func(err error) {
					innerMostRaised = true
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
}

func TestPanicNoErrorHandle(test *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			test.Errorf("The code did not panic")
		}
	}()
	Do(func() (string, error) {
		results := Do(func() (string, error) {
			results := Do(func() (string, error) {
				results := Do(func() (string, error) {
					return "", errors.New("something has gone wrong")
				})
				return results, nil
			})
			return results, nil
		})
		return results, nil
	})
}
