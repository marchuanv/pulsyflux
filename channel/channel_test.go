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
	var chl Chl[msgContract]
	var chlSubA ChlSub[msgContract]
	var chlSubB ChlSub[msgContract]
	defer (func() {
		err := recover()
		if err == nil {
			chl.Close()
		} else {
			test.Log(err)
			test.Fail()
		}
	})()
	msgA := &msgA{}
	msgB := &msgB{}

	chl = GetChl[msgContract](uuid.NewString())
	chlSubA = GetChlSub[msgContract](uuid.NewString())
	chlSubB = GetChlSub[msgContract](uuid.NewString())

	chl.Open()
	subReceivedCount := 0
	chlSubA.Subscribe(chl, func(msg msgContract) {
		subReceivedCount++
		if msg != msgA {
			test.Log("expected recevied msg to match either msgA or msgB")
			test.Fail()
		}
	})
	chl.Publish(msgA)
	chlSubA.Unsubscribe(chl)
	chlSubB.Subscribe(chl, func(msg msgContract) {
		subReceivedCount++
		if msg != msgB {
			test.Log("expected recevied msg to match either msgA or msgB")
			test.Fail()
		}
	})
	chl.Publish(msgB)
	chlSubB.Unsubscribe(chl)
	time.Sleep(1000 * time.Millisecond)
	if subReceivedCount != 2 {
		test.Logf("expected two subscriptions, received %d", subReceivedCount)
		test.Fail()
	}
}
