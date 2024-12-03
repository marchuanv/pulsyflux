package task

type tskCtx struct {
	curTsk *tskLink
	ch     Channel
}

type TaskCtx interface {
	Do(doFunc func(channel Channel), errorFuncs ...func(err error, channel Channel)) Channel
}

func NewTskCtx() TaskCtx {
	rootTsk := newTskLink()
	rootTsk.isRoot = true
	return newTskCtx(rootTsk)
}

func newTskCtx(tLink *tskLink) *tskCtx {
	return &tskCtx{tLink, &chnl{}}
}
