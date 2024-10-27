package msgbus

import (
	"runtime"
	"testing"
	"time"
)

const (
	MESSAGE_BUS MsgSubId = "90c305a6-c55f-412e-a49e-58bab71688b0"
)

func TestCreateChannel(test *testing.T) {
	msgBusCh := New(MESSAGE_BUS)
	msgBusCh.Open()
	expectedMsg := NewMessage("Hello World")
	msgBusCh.Publish(expectedMsg)
	receivedMsg := msgBusCh.Subscribe()
	if receivedMsg == expectedMsg {
		test.Fail()
	}
	if receivedMsg.GetId() != expectedMsg.GetId() {
		test.Fail()
	}
	msgBusCh.Close()
	runtime.GC()
	time.Sleep(5 * time.Second)
}
