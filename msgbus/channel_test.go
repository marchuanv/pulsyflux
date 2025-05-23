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
	BootStrap()
	msgBusCh := New(MESSAGE_BUS)
	msgBusCh.Open()
	expectedMsg := NewMessage("Hello World")
	msgBusCh.Publish(expectedMsg)
	receivedMsg := msgBusCh.Subscribe()
	if receivedMsg != expectedMsg {
		test.Log("expected message received from subscription to be the same as the original message published.")
		test.FailNow()
	}
	if receivedMsg.GetId() != expectedMsg.GetId() {
		test.Fail()
	}
	msgBusCh.Close()
	runtime.GC()
	time.Sleep(5 * time.Second)
}
