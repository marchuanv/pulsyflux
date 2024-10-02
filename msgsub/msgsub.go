package msgsub

import (
	"pulsyflux/message"
)

type MessageSubscription struct{}

func New(channel string) (MessageSubscription, error) {
	msg := message.Message{}
	return MessageSubscription{}, nil
}
