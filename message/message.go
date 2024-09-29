package message

import (
	"fmt"
	"uuid"
)

type Message struct {
	*uuid.UUID
	text string
}

func New(msgText ...string) *Message {
	uuidPtr := uuid.New(msgText[0])
	return &Message{
		UUID: uuidPtr,
		text: msgText[0],
	}
}
func (msg *Message) ToString() (string, error) {
	uuidStr, err := msg.UUID.ToString()
	if err != nil {
		return "", err
	}
	msgStr, err := msg.ToString()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("{\"Id\":\"%s\",\"message\":\"%s\"}", uuidStr, msgStr), nil
}
