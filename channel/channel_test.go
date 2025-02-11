package channel

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type msgImpA struct{}

type msgImpB struct{}

func (imp *msgImpA) helloWorld(msg string) {
}

func (imp *msgImpB) helloWorld(msg string) {
}

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
	defer (func() {
		err := recover()
		if err != nil {
			test.Log(err)
			test.Fail()
		}
	})()
	exMsgContent1 := &msgImpA{}
	exMsgContent2 := &msgImpB{}
	subId := SubId(uuid.New())
	chnlId := ChnlId(uuid.New())
	OpenChnl(chnlId)
	subReceivedCount := 0
	Subscribe(subId, chnlId, func(msgContent msgContract) {
		subReceivedCount++
		if msgContent != exMsgContent1 {
			if msgContent != exMsgContent2 {
				test.Log("expected all messages")
				test.Fail()
			}
		}
		if msgContent != exMsgContent2 {
			if msgContent != exMsgContent1 {
				test.Log("expected all messages")
				test.Fail()
			}
		}
	})
	Publish(chnlId, exMsgContent1)
	Publish(chnlId, exMsgContent2)
	time.Sleep(1000 * time.Millisecond)
	Unsubscribe(subId, chnlId)
	CloseChnl(chnlId)
	if subReceivedCount != 2 {
		test.Logf("expected only two subscriptions to be resolved, received %d", subReceivedCount)
		test.Fail()
	}
}

func TestChnlUnsubscribe(test *testing.T) {
	defer (func() {
		err := recover()
		if err != nil {
			test.Log(err)
			test.Fail()
		}
	})()
	exMsgContent1 := &msgImpA{}
	exMsgContent2 := &msgImpB{}
	subId := SubId(uuid.New())
	chnlId := ChnlId(uuid.New())
	OpenChnl(chnlId)
	subReceivedCount := 0
	Subscribe(subId, chnlId, func(msgContent msgContract) {
		subReceivedCount++
		if msgContent != exMsgContent1 {
			if msgContent != exMsgContent2 {
				test.Log("expected all messages")
				test.Fail()
			}
		}
		if msgContent != exMsgContent2 {
			if msgContent != exMsgContent1 {
				test.Log("expected all messages")
				test.Fail()
			}
		}
	})
	Publish(chnlId, exMsgContent1)
	Publish(chnlId, exMsgContent2)
	time.Sleep(1000 * time.Millisecond)
	Unsubscribe(subId, chnlId)
	if subReceivedCount != 2 {
		test.Logf("expected only two subscriptions to be resolved, received %d", subReceivedCount)
		test.Fail()
	}
	CloseChnl(chnlId)
}
