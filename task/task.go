package task

import (
	"reflect"
	"sync"
)

var allFailures []reflect.Value
var mu sync.Mutex

func Do[T any, T2 any](success func() (T, T2, error), failures ...func(err error, params ...T2)) T {
	shift[T](failures...)
	result, param, err := success()
	failure := pop[T, T2]()
	if err == nil {
		for failure != nil {
			failure = pop[T, T2]()
		}
	} else {
		errHandled := false
		for failure != nil {
			failure(err, param)
			errHandled = true
			failure = pop[T, T2]()
		}
		if !errHandled {
			panic(err)
		}
	}
	return result
}

func pop[T any, T2 any]() func(err error, params ...T2) {
	if len(allFailures) > 0 {
		mu.Lock()
		dqFailure := allFailures[len(allFailures)-1]
		allFailures = allFailures[:len(allFailures)-1]
		mu.Unlock()
		return dqFailure.Interface().(func(err error, params ...T2))
	}
	return nil
}

func shift[T any, T2 any](failures ...func(err error, params ...T2)) {
	mu.Lock()
	for _, failure := range failures {
		value := reflect.Indirect(reflect.ValueOf(failure))
		allFailures = append(allFailures, value)
	}
	mu.Unlock()
}
