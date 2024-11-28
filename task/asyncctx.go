package task

func (taskCtx *tskCtx[T1, T2]) DoLater(doFunc func(input T1) T2, receiveFunc func(results T2, input T1), errorFuncs ...func(err error, input T1) T1) {
	if receiveFunc == nil {
		panic("receive function is nil")
	}
	var errorFunc func(err error, errorParam T1) T1
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	var tLink *tskLink[T1, T2]
	tLinkByDoFuncCall := taskCtx.root.findNode(DoFunc)
	tLinkByRecvFuncCall := taskCtx.root.findNode(ReceiveFunc)
	tLinkByErrorFuncCall := taskCtx.root.findNode(ErrorFunc)
	if tLinkByErrorFuncCall != nil {
		tLinkByErrorFuncCall.newChildTsk(taskCtx.root.input)
		tLink = tLinkByErrorFuncCall
	} else if tLinkByRecvFuncCall != nil {
		tLinkByRecvFuncCall.newChildClnTsk()
		tLink = tLinkByRecvFuncCall
	} else if tLinkByDoFuncCall != nil {
		tLinkByDoFuncCall.newChildClnTsk()
		tLink = tLinkByDoFuncCall
	} else {
		tLink = taskCtx.root
	}
	tLink = tLink.getLeafNode() //Most recent child node created without children
	tLink.doFunc = doFunc
	tLink.receiveFunc = receiveFunc
	tLink.errorFunc = errorFunc
	tLink.run()
}
