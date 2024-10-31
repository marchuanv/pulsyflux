package util

var failureQueue = make([]func(err error), 0)

func Do[T any](success func() (T, error), failures ...func(err error)) T {
	queueFailures(failures...)
	result, err := success()
	errHandled := false
	failure := dequeueFailure()
	if err == nil {
		for failure != nil {
			failure = dequeueFailure()
		}
	} else {
		for failure != nil {
			if !errHandled {
				errHandled = true
				failure(err)
			}
			failure = dequeueFailure()
		}
		if !errHandled {
			panic(err)
		}
	}
	return result
}

func dequeueFailure() func(err error) {
	if len(failureQueue) > 0 {
		dqFailure := failureQueue[0]
		failureQueue = failureQueue[:0]
		return dqFailure
	}
	return nil
}

func queueFailures(failures ...func(err error)) {
	failureQueue = append(failureQueue, failures...)
}
