package msgbus

import (
	"pulsyflux/message"
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

func newMessage(test *testing.T, msgText string) *message.Message {
	msg, err := message.New(msgText)
	if err == nil {
		return msg
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

func sendMsg(test *testing.T, msgBus *MessageBus, msg *message.Message) {
	err = msgBus.Send("http://localhost:3000/subscriptions/testchannel", msg)
	if err == nil {
		return
	}
	test.Fatalf("exiting test") // Exits the program
}

func TestMessageBus(test *testing.T) {
	msgBus := newMsgBus(test, "locahost", 3000)
	msgToSend := newMessage(test, "Hello World")
	startMsgBus(test, msgBus)

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
