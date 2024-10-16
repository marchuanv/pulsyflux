package msgbus

import (
	"testing"
)

func newMsgBus(test *testing.T, host string, port int) *MessageBus {
	msgBus, err := New(host, port)
	if err == nil {
		return msgBus
	}
	test.Fatalf("exiting test") // Exits the program
	return nil
}

func startMsgBus(test *testing.T, msgBus *MessageBus) {
	err := msgBus.Start()
	if err == nil {
		return
	}
	test.Fatalf("exiting test") // Exits the program
}

func TestStartTwoMessageBusOnSamePort(t *testing.T) {
	msgBus, err := New("localhost", 4000)
	if err != nil {
		t.Error(err)
		return
	}
	err = msgBus.Start()
	if err != nil {
		t.Error(err)
		return
	}
	msgBus2, err := New("localhost", 4000)
	if err != nil {
		t.Error(err)
		return
	}
	err = msgBus2.Start()
	if err == nil {
		msgBus.Stop()
		t.Errorf("expected an error when starting the second message bus on the same port")
	} else {
		msgBus2.Stop()
		t.Log(err.Error())
	}
}
