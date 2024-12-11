package channel

import (
	"testing"
)

type someStruct struct{}

func TestStringChnl(test *testing.T) {
	ch := NewChnl()
	ch.Send(NewChnlMsg("HelloWorld"))
	msg, _ := ch.Message().Content()
	if msg != "HelloWorld" {
		test.Fail()
	}
}
func TestIntChnl(test *testing.T) {
	ch := NewChnl()
	ch.Send(NewChnlMsg(12345))
	msg, _ := ch.Message().Content()
	if msg != 12345 {
		test.Fail()
	}
}

func TestPtrChnl(test *testing.T) {
	ptr := &someStruct{}
	ch := NewChnl()
	ch.Send(NewChnlMsg(ptr))
	msg, _ := ch.Message().Content()
	if msg != ptr {
		test.Fail()
	}
}

func TestChnlMsgTimeout(test *testing.T) {
	ptr := &someStruct{}
	ch := NewChnl()
	msg, _ := ch.Message().Content()
	if msg != ptr {
		test.Fail()
	}
}
