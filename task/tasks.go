package task

import "pulsyflux/channel"

func (taskCtx *tskCtx) Do(doFunc func(channel channel.Channel), errorFuncs ...func(err error, channel channel.Channel)) channel.Channel {
	var errorFunc func(err error, channel channel.Channel)
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
	if tLink.isRoot {
		taskCtx.ch = tLink.channel
	}
	return taskCtx.ch
}
