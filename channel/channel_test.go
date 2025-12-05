package channel

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type msgA struct{}

type msgB struct{}

func (imp *msgA) helloWorld(msg string) {}

func (imp *msgB) helloWorld(msg string) {}

type msgContract interface{ helloWorld(msg string) }

func TestChnlErrorConvert(test *testing.T) {
	chnlErr := newChnlError("hello world")
	_, canConv := isChnlError(chnlErr)
	if !canConv {
		test.Fail()
	}
	someErr := errors.New("hello world")
	_, canConv = isChnlError(someErr)
	if canConv {
		test.Fail()
	}
	_, canConv = isError[error](chnlErr)
	if !canConv {
		test.Fail()
	}
	_, canConv = isError[error](someErr)
	if !canConv {
		test.Fail()
	}
	someErrStr := "hello world"
	_, canConv = isError[error](someErrStr)
	if canConv {
		test.Fail()
	}
}

func TestChnlSubscribe(test *testing.T) {
	var chlId ChlId
	var chlSubAId ChlSubId
	var chlSubBId ChlSubId
	defer (func() {
		err := recover()
		if err == nil {
			chlId.CloseChnl()
		} else {
			test.Log(err)
			test.Fail()
		}
	})()
	msgA := &msgA{}
	msgB := &msgB{}

	chlId = NewChl(uuid.NewString())
	chlSubAId = NewChlSub(uuid.NewString())
	chlSubBId = NewChlSub(uuid.NewString())

	chlId.OpenChnl()
	subReceivedCount := 0
	chlSubAId.Subscribe(chlId, func(msg any) {
		subReceivedCount++
		if msg != msgA {
			test.Log("expected recevied msg to match either msgA or msgB")
			test.Fail()
		}
	})
	chlId.Publish(msgA)
	chlSubAId.Unsubscribe(chlId)
	chlSubBId.Subscribe(chlId, func(msg any) {
		subReceivedCount++
		if msg != msgB {
			test.Log("expected recevied msg to match either msgA or msgB")
			test.Fail()
		}
	})
	chlId.Publish(msgB)
	chlSubBId.Unsubscribe(chlId)
	time.Sleep(1000 * time.Millisecond)
	if subReceivedCount != 2 {
		test.Logf("expected only two subscriptions to be resolved, received %d", subReceivedCount)
		test.Fail()
	}
}
