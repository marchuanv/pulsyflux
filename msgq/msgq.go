package msgq

import (
	"errors"
	"fmt"
	"pulsyflux/message"
	"pulsyflux/util"
	"sync"
)

type MsgQueue struct {
	channel  string
	mu       *sync.Mutex
	messages []*message.Message
}

var queues map[string]*MsgQueue

func Get(channel string) (*MsgQueue, error) {
	if len(channel) == 0 {
		return nil, errors.New("the channel argument is an empty string")
	}
	if !util.IsValidUUID(channel) {
		return nil, errors.New("the channel argument is not a uuid")
	}
	if queues == nil {
		queues = make(map[string]*MsgQueue)
	}
	queue, exists := queues[channel]
	if exists {
		return queue, nil
	}
	queue = &MsgQueue{
		channel,
		&sync.Mutex{},
		[]*message.Message{},
	}
	queues[channel] = queue
	return queue, nil
}

func (msgq *MsgQueue) Dequeue() *message.Message {
	if len(msgq.messages) > 0 {
		msgq.mu.Lock()
		defer msgq.mu.Unlock()
		msg := msgq.messages[0]
		msgq.messages = msgq.messages[1:]
		return msg
	}
	return nil
}

func (msgq *MsgQueue) Enqueue(message *message.Message) error {
	if msgq.channel != message.Channel() {
		return errors.New("failed to enqueue message from a different channel")
	}
	msgq.mu.Lock()
	defer msgq.mu.Unlock()
	msgq.messages = append(msgq.messages, message)
	fmt.Printf("message is queued.\n")
	return nil
}
