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
	uuidPtr, exists := _messages[idStr]
	if exists {
		return uuidPtr, nil
	}
	_messages[idStr] = &Message{idStr, msgText}
	return _messages[idStr], nil

}
func v5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}
