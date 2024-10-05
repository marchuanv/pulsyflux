package msgbus

import (
	"pulsyflux/message"
	"testing"
)

func TestMessageBus(t *testing.T) {
	msgBus, err := New("localhost", 3000)
	if err != nil {
		t.Error(err)
		return
	}
	msgToSend, err := message.New("Hello World")
	if err != nil {
		t.Error(err)
		return
	}
	err = msgBus.Start()
	if err != nil {
		t.Error(err)
		return
	}
	err = msgBus.Send("http://localhost:3000/subscriptions/testchannel", msgToSend)
	if err != nil {
		t.Error(err)
		return
	}
	msgReceived := msgBus.Dequeue()
	if msgReceived == nil {
		t.Error("MessageBus: expected a message to be dequeued")
	} else {
		t.Logf("MessageId:%s, MessageText:%s", msgReceived.Id, msgReceived.Text)
	}
	msgBus.Stop()
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
