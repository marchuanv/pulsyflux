package task

func (taskCtx *tskCtx) Do(doFunc func(channel Channel), errorFuncs ...func(err error, channel Channel)) Channel {
	var errorFunc func(err error, channel Channel)
	if len(errorFuncs) > 0 {
		errorFunc = errorFuncs[0]
	}
	tLink := taskCtx.curTsk.getLeafNode(ErrorFunc, DoFunc)
	switch tLink.funcCallstack.Peek() {
	case ErrorFunc:
		tLink.newChildTsk()
	case DoFunc:
		tLink.newChildTskClone()
	}
	tLink = tLink.getLeafNode() //Most recent child node created without children
	tLink.doFunc = doFunc
	tLink.errorFunc = errorFunc
	tLink.run()
	return tLink.channel
}
