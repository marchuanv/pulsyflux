package util

type Result[T any] struct {
	Output T
}

func Do[T any](exitOnError bool, success func() (*Result[T], error), failure ...func(err error)) *Result[T] {
	if len(failure) > 1 {
		panic("only one failure function is supported.")
	}
	var result *Result[T]
	var err error
	if exitOnError {
		result, err = success()
		if err != nil {
			panic(err)
		}
	} else {
		if len(failure) == 0 {
			panic("failure handle function required when exit on error is false")
		} else {
			result, err = success()
			if err != nil {
				failure[0](err)
			}
		}
	}
	return result
}
