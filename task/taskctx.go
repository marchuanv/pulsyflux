package task

type tskCtx[T1 any, T2 any] struct {
	root *tskLink[T1, T2]
}

type TaskCtx[T1 any, T2 any] interface {
	DoNow(doFunc func(input T1) T2, errorFuncs ...func(err error, input T1) T1) T2
	DoLater(doFunc func(input T1) T2, receiveFunc func(results T2, input T1), errorFuncs ...func(err error, input T1) T1)
}

func NewTskCtx[T1 any, T2 any](input T1) TaskCtx[T1, T2] {
	rootTsk := newTskLink[T1, T2](input)
	rootTsk.isRoot = true
	return newTskCtx(rootTsk)
}

func newTskCtx[T1 any, T2 any](tLink *tskLink[T1, T2]) *tskCtx[T1, T2] {
	return &tskCtx[T1, T2]{tLink}
}
