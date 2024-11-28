package task

func (taskCtx *tskCtx[T1, T2]) DoNow(doFunc func(input T1) T2, errorFuncs ...func(err error, input T1) T1) T2 {
	var errorFunc func(err error, input T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	tLink := taskCtx.root.getLeafNode(ErrorFunc, DoFunc)
	switch tLink.funcCallstack.Peek() {
	case ErrorFunc:
		tLink.newChildTsk(taskCtx.root.input)
	case DoFunc:
		tLink.newChildTskClone()
	}
	tLink = tLink.getLeafNode() //Most recent child node created without children
	tLink.doFunc = doFunc
	tLink.errorFunc = errorFunc
	tLink.run()
	return tLink.result
}
