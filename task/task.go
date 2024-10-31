package task

import (
	"reflect"
	"sync"
)

var allFailures []reflect.Value
var mu sync.Mutex

func Do[T any, T2 any](success func() (T, error), failures ...func(err error, params ...T) T2) (T, T2) {
	enqueue[T](failures...)
	result, err := success()
	var failureResult T2
	failure := dequeue[T, T2]()
	if err == nil {
		for failure != nil {
			failure = dequeue[T, T2]()
		}
	} else {
		errHandled := false
		for failure != nil {
			if !errHandled {
				errHandled = true
				failureResult = failure(err, result)
			}
			failure = dequeue[T, T2]()
		}
		if !errHandled {
			panic(err)
		}
	}
	return result, failureResult
}

func dequeue[T any, T2 any]() func(err error, params ...T) T2 {
	if len(allFailures) > 0 {
		mu.Lock()
		dqFailure := allFailures[0]
		allFailures = allFailures[:0]
		mu.Unlock()
		return dqFailure.Interface().(func(err error, params ...T) T2)
	}
	return nil
}

func enqueue[T any, T2 any](failures ...func(err error, params ...T) T2) {
	mu.Lock()
	for _, failure := range failures {
		value := reflect.Indirect(reflect.ValueOf(failure))
		allFailures = append(allFailures, value)
	}
	mu.Unlock()
}
