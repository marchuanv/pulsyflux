package task

import (
	"pulsyflux/sliceext"
	"sync"
	"time"

	"github.com/google/uuid"
)

type chnlState string
type Channel interface {
	Read(receiveData func(data any))
	Write(data any)
}
type chnl struct {
	Id        string
	state     chnlState
	mu        sync.Mutex
	dataQueue sliceext.Queue[*chnlData]
}

type chnlData struct {
	Id   string
	data any
}

const (
	Open   chnlState = "aa901325-9f2b-4678-8208-7cc979cbd70f"
	Closed chnlState = "c4b12101-3c6a-46c2-b7cb-8b4d77646ce9"
	Locked chnlState = "4cf941ac-2773-4786-aeeb-c3ef147a9968"
)

func newChl() *chnl {
	return &chnl{
		uuid.NewString(),
		Open,
		sync.Mutex{},
		sliceext.Queue[*chnlData]{},
	}
}

func newChlData(Id string, data any) *chnlData {
	return &chnlData{Id, data}
}

func (ch *chnl) Read(receiveData func(data any)) {
	defer ch.unlock()
	ch.lock()
	for ch.dataQueue.Peek() != nil {
		chnlData := ch.dataQueue.Dequeue()
		if chnlData.Id == ch.Id {
			receiveData(chnlData.data)
		}

	}
}

func (ch *chnl) Write(data any) {
	defer ch.unlock()
	ch.lock()
	ch.dataQueue.Enqueue(newChlData(ch.Id, data))
}

func (ch *chnl) close() {
	ch.lock()
	ch.state = Closed
	ch.unlock()
}

func (ch *chnl) lock() bool {
	if ch.getState() == Closed {
		panic("channel is closed")
	}
	for ch.getState() == Locked {
		time.Sleep(1 * time.Second)
	}
	ch.state = Locked
	return true
}

func (ch *chnl) unlock() {
	if ch.getState() == Locked {
		ch.state = Open
	} else {
		panic("channel is not locked")
	}
}

func (ch *chnl) getState() chnlState {
	defer ch.mu.Unlock()
	ch.mu.Lock()
	return ch.state
}

func (ch *chnl) setState(state chnlState) {
	defer ch.mu.Unlock()
	ch.mu.Lock()
	ch.state = state
}
