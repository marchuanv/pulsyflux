package util

func Do[T any](exitOnError bool, success func() (T, error), failure ...func(err error)) T {
	if len(failure) > 1 {
		panic("only one failure function is supported.")
	}
	var result T
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
