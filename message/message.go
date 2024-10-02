package message

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"

	"github.com/google/uuid"
)

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

// serialise message
func (msg *Message) Serialise() (string, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(msg)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

func DeserialiseMessage(msgStr string) (Message, error) {
	msg := Message{}
	by, err := base64.StdEncoding.DecodeString(msgStr)
	if err != nil {
		return msg, err
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&msg)
	if err != nil {
		return msg, err
	}
	return msg, nil
}

func v5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}
