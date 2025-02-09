package channel

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type someStruct struct{}

type someStruct2 struct{}

func TestStringChnl(test *testing.T) {
	ch := NewChnl()
	ch.send(newChnlMsg("HelloWorld"))
	msg, _ := ch.next().content()
	if msg != "HelloWorld" {
		test.Fail()
	}
}
func TestIntChnl(test *testing.T) {
	ch := NewChnl()
	ch.send(newChnlMsg(12345))
	msg, _ := ch.next().content()
	if msg != 12345 {
		test.Fail()
	}
}

func TestPtrChnl(test *testing.T) {
	ptr := &someStruct{}
	ch := NewChnl()
	ch.send(newChnlMsg(ptr))
	msg, _ := ch.next().content()
	if msg != ptr {
		test.Fail()
	}
}

func TestChnlMsgTimeout(test *testing.T) {
	defer (func() {
		err := recover()
		if err == nil {
			test.Fail()
		}
	})()
	ch := NewChnl()
	ch.next().content()
}

func TestChnlSubscribe(test *testing.T) {
	defer (func() {
		err := recover()
		if err != nil {
			test.Log(err)
			test.Fail()
		}
	})()
	exMsgContent1 := &someStruct{}
	exMsgContent2 := &someStruct2{}
	ch := NewChnl()
	subId := uuid.New()
	Subscribe(subId, ch, func(content *someStruct) {
		if content == exMsgContent1 {
			test.Log("expected content to match exMsgContent1")
			test.Fail()
		}
	})
	Publish(ch, exMsgContent1)
	Publish(ch, exMsgContent2)
	time.Sleep(1000 * time.Millisecond)
}
