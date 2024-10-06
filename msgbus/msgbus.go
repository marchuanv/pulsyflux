package msgbus

import (
	"fmt"
	"pulsyflux/message"
	"sync"
)

type MsgBus struct {
	mu    *sync.Mutex
	queue *MsgQueue
	host  string
	port  int
}

func MessageBus(host string, port int) (*MsgBus, error) {
	msgBus := &MsgBus{
		&sync.Mutex{},
		nil,
		host,
		port,
	}
	msgQ, err := MessageQueue(msgBus)
	if err != nil {
		return nil, err
	}
	msgBus.queue = msgQ
	return msgBus, nil
}

func (msgBus *MsgBus) Dequeue() *message.Message {
	if len(msgBus.queue.messages) > 0 {
		msgBus.mu.Lock()
		defer msgBus.mu.Unlock()
		msg := msgBus.queue.messages[0]
		messages := msgBus.queue.messages[1:]
		msgBus.queue = nil
		msgBus.queue.messages = messages
		return msg
	}
	return nil
}

func (msgBus *MsgBus) Enqueue(message *message.Message) {
	msgBus.mu.Lock()
	defer msgBus.mu.Unlock()
	msgBus.queue.messages = append(msgBus.queue.messages, message)
	fmt.Printf("message is queued.\n")
}
