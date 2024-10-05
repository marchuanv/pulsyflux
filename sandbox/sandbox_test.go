package msgbus

import (
	"fmt"
	"pulsyflux/message"
	"testing"
	"time"
)

type MessageBus struct {
	queue []*message.Message
}

func (msgBus *MessageBus) Dequeue() *message.Message {
	msg, s2 := msgBus.queue[0], msgBus.queue[1:]
	msgBus.queue = s2
	return msg
}
func (msgBus *MessageBus) Enqueue(message *message.Message) {
	msgBus.queue = append(msgBus.queue, message)
}

func TestThings(t *testing.T) {
	msgBusQueue := []*message.Message{}
	msgBus := MessageBus{}
	msgBus.queue = msgBusQueue
	newMsg := message.Message{}
	go (func() {
		msgBus.Enqueue(&newMsg)
		fmt.Printf("message was queued.\n\r")
	})()
	time.Sleep(1 * time.Second)
	go (func() {
		msg := msgBus.Dequeue()
		serialisedMsg, _ := msg.Serialise()
		fmt.Printf("message was dequeued: %s", serialisedMsg)
	})()
	time.Sleep(2 * time.Second)
}
