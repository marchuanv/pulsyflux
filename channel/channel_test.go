package channel

import (
	"testing"
)

func TestChnl(test *testing.T) {
	ch := NewChnl[string]()
	ch.Send(newChnlMsg("HelloWorld"))
	msg := ch.Receive()
	if msg.Data() != "HelloWorld" {
		test.Fail()
	}
}
