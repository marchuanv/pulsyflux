package task

import "pulsyflux/channel"

type tskCtx struct {
	curTsk *tskLink
	ch     channel.Channel
}

type TaskCtx interface {
	Do(doFunc func(channel channel.Channel), errorFuncs ...func(err error, channel channel.Channel)) channel.Channel
}

func NewTskCtx() TaskCtx {
	rootTsk := newTskLink()
	rootTsk.isRoot = true
	return newTskCtx(rootTsk)
}

func newTskCtx(tLink *tskLink) *tskCtx {
	return &tskCtx{tLink, nil}
}
