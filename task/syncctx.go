package task

func (taskCtx *tskCtx[T1, T2]) DoNow(doFunc func(input T1) T2, errorFuncs ...func(err error, input T1) T1) T2 {
	tLink := taskCtx.root
	tLink = tLink.getSyncLeafNode()
	if taskCtx.syncCallCount > 0 { //if leaf node is not root node
		tLink.spawnChildSyncTskLink()
		tLink = tLink.getSyncLeafNode()
	}
	var errorFunc func(err error, input T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	tLink.doFunc = doFunc
	tLink.errorFunc = errorFunc
	taskCtx.syncCallCount += 1
	taskCtx.doSync(tLink)
	return tLink.result
}

func (taskCtx *tskCtx[T1, T2]) doSync(tLink *tskLink[T1, T2]) {
	defer tLink.unlinkSync()
	tLink.run()
	if tLink.errOnHandle != nil {
		panic(tLink.errOnHandle)
	}
	if tLink.err != nil {
		if tLink.parent == nil {
			if !tLink.errorHandled {
				panic(tLink.err)
			}
		}
	}
}
