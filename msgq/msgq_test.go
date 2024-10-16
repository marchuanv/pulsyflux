package msgq

import (
	"pulsyflux/message"
	"testing"
	"time"
)

func getMsgQ(test *testing.T, channel string) *MsgQueue {
	msgQ, err := Get(channel)
	if err == nil {
		return msgQ
	}
	test.Fatalf("failed to create message queue.%s", err) // Exits the program
	return nil
}

func newMsg(test *testing.T, channel string, fromAddress string, toAddress string, text string) message.Message {
	msg, err := message.New(channel, fromAddress, toAddress, text)
	if err == nil {
		return msg
	}
	test.Fatalf("failed to create message.%s", err) // Exits the program
	return message.Message{}
}

func TestGetExistingQueue(test *testing.T) {

	channel1 := "ae170138-6233-4794-be32-0559ff4e4498"
	msgQ1 := getMsgQ(test, channel1)
	msgQ2 := getMsgQ(test, channel1)

	if msgQ1 != msgQ2 {
		test.Fatal("expected msgq pointers to match")
	}
}

func TestMessageEnqueueFromDifferentChannel(test *testing.T) {

	channel1 := "509f2895-02d7-4a39-aa0b-718b0bfedffc"
	msgQ1 := getMsgQ(test, channel1)
	msq1 := newMsg(test, channel1, "localhost:3000", "localhost:3000", "hello World from msg1")
	msgQ1.Enqueue(&msq1)

	channel2 := "a2441563-d429-4e51-8aa0-8eb233039c82"
	msq2 := newMsg(test, channel2, "localhost:3000", "localhost:3000", "hello World from msg2")
	err := msgQ1.Enqueue(&msq2)
	if err == nil {
		test.Fatal("expected message enqueue to fail.")
	}
}

func TestSameChannelMessagesDequeuedMessagesAreNotTheSame(test *testing.T) {

	channel := "5ae92e90-ea03-45a4-9a18-f218427eaa57"
	msgQ1 := getMsgQ(test, channel)

	msq1 := newMsg(test, channel, "localhost:3000", "localhost:3000", "hello World")
	err := msgQ1.Enqueue(&msq1)
	if err != nil {
		test.Error(err)
	}
	msq2 := newMsg(test, channel, "localhost:3000", "localhost:3000", "hello World")
	err = msgQ1.Enqueue(&msq2)
	if err != nil {
		test.Error(err)
	}

	deqMsg1 := msgQ1.Dequeue()
	deqMsg2 := msgQ1.Dequeue()

	if deqMsg1 == deqMsg2 {
		test.Error("dequeued messages are not the same")
	}
}

func TestSameChannelSameEnqueuedMessageDequeuedThreadSafe(test *testing.T) {

	channel := "5ae92e90-ea03-45a4-9a18-f218427eaa57"
	msgQ1 := getMsgQ(test, channel)

	msq1 := newMsg(test, channel, "localhost:3000", "localhost:3000", "hello World")
	err := msgQ1.Enqueue(&msq1)
	if err != nil {
		test.Error(err)
	}

	var dequeuedMsg1 *message.Message
	var dequeuedMsg2 *message.Message
	go (func() {
		dequeuedMsg1 = msgQ1.Dequeue()
	})()
	go (func() {
		dequeuedMsg2 = msgQ1.Dequeue()
	})()

	time.Sleep(2 * time.Second)

	if dequeuedMsg1 != nil && dequeuedMsg1 != &msq1 {
		test.Error("enqueued message was not the same message that was dequeued")
	}

	if dequeuedMsg2 != nil && dequeuedMsg2 != &msq1 {
		test.Error("enqueued message was not the same message that was dequeued")
	}

	if dequeuedMsg1 == dequeuedMsg2 {
		test.Error("dequeued messages should not be the same, both dequeue routines dequeued the same message")
	}
}
