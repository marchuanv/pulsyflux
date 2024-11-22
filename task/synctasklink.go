package task

import "github.com/google/uuid"

func (tLink *tskLink[T1, T2]) getSyncLeafNode() *tskLink[T1, T2] {
	for _, child := range tLink.syncChildren {
		return child.getSyncLeafNode()
	}
	return tLink
}

func (tLink *tskLink[T1, T2]) spawnChildSyncTskLink() {
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
	tLink.syncChildren[newTskLink.Id] = newTskLink
}

func (tLink *tskLink[T1, T2]) unlinkSync() {
	if len(tLink.syncChildren) > 0 {
		panic("fatal error node has children")
	}
	if tLink.parent != nil {
		delete(tLink.parent.syncChildren, tLink.Id)
		tLink.parent.unlinkedChildren[tLink.Id] = tLink
	}
}
