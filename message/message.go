package message

import "github.com/google/uuid"

type Message struct {
	Id   string
	Text string
}

var _messages = map[string]*Message{}

func New(msgText string) (*Message, error) {
	id := v5UUID(msgText)
	idStr := id.String()
	msg, exists := _messages[idStr]
	if exists {
		return msg, nil
	}
	//use pointers the hastable grows and structs may be moved around which would not be a the same address anymore
	_messages[idStr] = &Message{idStr, msgText}
	return New(msgText)
}
func v5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}
