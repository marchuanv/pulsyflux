package task

func (taskCtx *tskCtx[T1, T2]) DoLater(doFunc func(input T1) T2, receiveFunc func(results T2, input T1), errorFuncs ...func(err error, input T1) T1) {
	if receiveFunc == nil {
		panic("receive function is nil")
	}
	var errorFunc func(err error, errorParam T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	tLink := taskCtx.root
	if taskCtx.asyncCallCount > 0 {
		taskCtx.root.spawnChildAsyncTskLink()
	}
	tLink = tLink.getAsyncLeafNode()
	tLink.doFunc = doFunc
	tLink.receiveFunc = receiveFunc
	tLink.errorFunc = errorFunc
	taskCtx.asyncCallCount += 1
	taskCtx.doAsync(tLink)
}

func (taskCtx *tskCtx[T1, T2]) doAsync(tLink *tskLink[T1, T2]) {
	go (func() {
		defer tLink.unlinkAsync()
		tLink.run()
		if tLink.errOnHandle != nil {
			panic(tLink.errOnHandle)
		}
		if tLink.err != nil {
			if !tLink.errorHandled {
				panic(tLink.err)
			}
		}
	})()
}
