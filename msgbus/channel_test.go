package msgbus

import (
	"runtime"
	"testing"
	"time"
)

const (
	MESSAGE_BUS MsgSubId = "90c305a6-c55f-412e-a49e-58bab71688b0"
)

func createChannel(test *testing.T, subId MsgSubId, exitOnError bool) *Channel {
	ch, err := New(subId)
	if err == nil {
		return ch
	}
	if exitOnError {
		test.Fatal(err) // Exits the program
	}
	return ch
}

func TestCreateChannel(test *testing.T) {
	msgBusCh := createChannel(test, MESSAGE_BUS, true)
	err := msgBusCh.Open()
	if err == nil {
		expectedMsg := createMessage(test, "Hello World", true)
		err := msgBusCh.Publish(expectedMsg)
		if err == nil {
			var receivedMsg Msg
			receivedMsg, err = msgBusCh.Subscribe()
			if err == nil {
				if receivedMsg == expectedMsg {
					test.Fail()
				}
				if receivedMsg.GetId() != expectedMsg.GetId() {
					test.Fail()
				}
			}
		} else {
			test.Fatal(err)
		}
	}
	msgBusCh.Close()
	runtime.GC()
	time.Sleep(5 * time.Second)
}
