package msgbus

import (
	"pulsyflux/message"
	"testing"
	"time"
)

func TestMessageBus(t *testing.T) {
	msgBus := New("localhost", 3000)
	msgBus.Start()
	msgToSend, err := message.New("Hello World")
	if err != nil {
		t.Errorf("CtorError: failed to create message")
		return
	}
	err = msgBus.Send("http://localhost:3000/subscriptions/testchannel", msgToSend)
	if err != nil {
		t.Errorf("CtorError: failed to create message")
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
	msgBus := New("localhost", 4000)
	msgBus.Start()
	msgBus2 := New("localhost", 4000)
	msgBus2.Start()
	msgBus.Stop()
	msgBus2.Stop()
	time.Sleep(10 * time.Second)
}
