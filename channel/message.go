package channel

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"pulsyflux/util"

	"github.com/google/uuid"
)

type Message struct {
	id          uuid.UUID
	text        string
	fromAddress MessageAddress
	toAddress   MessageAddress
}

type MessageAddress struct {
	Host string
	Port int
}

type message struct {
	Id          uuid.UUID
	Text        string
	FromAddress MessageAddress
	ToAddress   MessageAddress
}

func New(id uuid.UUID, fromAddress string, toAddress string, text string) (*Message, error) {
	if len(fromAddress) == 0 {
		return nil, errors.New("the fromAddress argument is an empty string")
	}
	if len(toAddress) == 0 {
		return nil, errors.New("the toAddress argument is an empty string")
	}
	if len(text) == 0 {
		return nil, errors.New("the msgText argument is an empty string")
	}
	fromHost, fromPort, fromErr := util.GetHostAndPortFromAddress(fromAddress)
	if fromErr != nil {
		return nil, fromErr
	}
	toHost, toPort, toErr := util.GetHostAndPortFromAddress(toAddress)
	if toErr != nil {
		return nil, toErr
	}
	newMsg := Message{
		id,
		text,
		MessageAddress{fromHost, fromPort}, //from
		MessageAddress{toHost, toPort},     //to
	}
	return &newMsg, nil
}

func (msg *Message) Id() uuid.UUID {
	return msg.id
}

func (msg *Message) Text() string {
	return msg.text
}

func (msg *Message) FromAddress() MessageAddress {
	return msg.fromAddress
}

func (msg *Message) ToAddress() MessageAddress {
	return msg.toAddress
}

func (msg *Message) serialise() (string, error) {
	msgToSerialise := message{
		msg.id,
		msg.text,
		msg.fromAddress,
		msg.toAddress,
	}
	buffer := bytes.Buffer{}
	e := gob.NewEncoder(&buffer)
	err := e.Encode(msgToSerialise)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func deserialise(serialised string) (*Message, error) {
	if len(serialised) == 0 {
		return nil, errors.New("the serialised argument is an empty string")
	}
	msg := message{}
	by, err := base64.StdEncoding.DecodeString(serialised)
	if err != nil {
		return nil, err
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&msg)
	if err == nil {
		fromAddress := fmt.Sprintf("%s:%d", msg.FromAddress.Host, msg.FromAddress.Port)
		toAddress := fmt.Sprintf("%s:%d", msg.ToAddress.Host, msg.ToAddress.Port)
		return New(msg.Id, fromAddress, toAddress, msg.Text)
	}
	return nil, err
}
