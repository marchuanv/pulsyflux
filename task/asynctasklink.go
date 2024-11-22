package task

import "github.com/google/uuid"

func (tLink *tskLink[T1, T2]) getAsyncLeafNode() *tskLink[T1, T2] {
	for _, child := range tLink.asyncChildren {
		return child.getAsyncLeafNode()
	}
	return tLink
}

func (tLink *tskLink[T1, T2]) spawnChildAsyncTskLink() {
	newTskLink := &tskLink[T1, T2]{
		uuid.NewString(),
		tLink.input,
		tLink.result,
		tLink.err,
		tLink.errorHandled,
		tLink.errOnHandle,
		tLink.doFunc,
		tLink.receiveFunc,
		tLink.errorFunc,
		tLink,
		map[string]*tskLink[T1, T2]{},
		map[string]*tskLink[T1, T2]{},
		map[string]*tskLink[T1, T2]{},
		map[callCxt]callState{},
		false,
		false,
	}
	tLink.asyncChildren[newTskLink.Id] = newTskLink
}

func (tLink *tskLink[T1, T2]) unlinkAsync() {
	if len(tLink.asyncChildren) > 0 {
		panic("fatal error node has children")
	}
	if tLink.parent != nil {
		delete(tLink.parent.asyncChildren, tLink.Id)
		tLink.parent.unlinkedChildren[tLink.Id] = tLink
	}
}
