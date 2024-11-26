package task

func (taskCtx *tskCtx[T1, T2]) DoNow(doFunc func(input T1) T2, errorFuncs ...func(err error, input T1) T1) T2 {
	var errorFunc func(err error, input T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	var tLink *tskLink[T1, T2]
	tLinkByDoFuncCall := taskCtx.root.getNodeBy(DoFunc)
	tLinkByErrorFuncCall := taskCtx.root.getNodeBy(ErrorFunc)
	if tLinkByErrorFuncCall != nil {
		tLinkByErrorFuncCall.newChildTsk(taskCtx.root.input)
		tLink = tLinkByErrorFuncCall
	} else if tLinkByDoFuncCall != nil {
		tLinkByDoFuncCall.newChildClnTsk()
		tLink = tLinkByDoFuncCall
	} else {
		tLink = taskCtx.root
	}
	tLink = tLink.getLeafNode() //Most recent child node created without children
	tLink.doFunc = doFunc
	tLink.errorFunc = errorFunc
	tLink.run()
	return tLink.result
}
