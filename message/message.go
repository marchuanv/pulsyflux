package message

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/google/uuid"
)

type Message struct {
	id          string
	text        string
	channel     string
	fromAddress MessageAddress
	toAddress   MessageAddress
}

type MessageAddress struct {
	Host string
	Port int
}

type serialisedMsg struct {
	Id          string
	Text        string
	Channel     string
	FromAddress MessageAddress
	ToAddress   MessageAddress
}

var _messages = map[string]*Message{}

func NewMessage(channel string, fromAddress string, toAddress string, text string) (*Message, error) {
	if len(text) == 0 {
		return nil, errors.New("the msgText argument is an empty string")
	}
	id := v5UUID(text)
	idStr := id.String()
	msg, exists := _messages[idStr]
	if exists {
		return msg, nil
	}
	if len(channel) == 0 {
		return nil, errors.New("the channel argument is an empty string")
	}
	if len(fromAddress) == 0 {
		return nil, errors.New("the fromAddress argument is an empty string")
	}
	if len(toAddress) == 0 {
		return nil, errors.New("the toAddress argument is an empty string")
	}
	fromHost, fromPort, fromErr := getHostAndPortFromAddress(fromAddress)
	if fromErr != nil {
		return nil, fromErr
	}
	toHost, toPort, toErr := getHostAndPortFromAddress(toAddress)
	if toErr != nil {
		return nil, toErr
	}
	//use pointers the hastable grows and structs may be moved around which would not be a the same address anymore
	_messages[idStr] = &Message{
		idStr,
		text,
		channel,
		MessageAddress{fromHost, fromPort}, //from
		MessageAddress{toHost, toPort},     //to
	}
	newMsg, err := NewMessage(channel, fromAddress, toAddress, text)
	return newMsg, err
}

func NewDeserialiseMessage(serialise string) (*Message, error) {
	if len(serialise) == 0 {
		return nil, errors.New("the channel argument is an empty string")
	}
	serMsg := serialisedMsg{}
	by, err := base64.StdEncoding.DecodeString(serialise)
	if err != nil {
		return nil, err
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&serMsg)
	if err == nil {
		fromAddress := fmt.Sprintf("%s:%d", serMsg.FromAddress.Host, serMsg.FromAddress.Port)
		toAddress := fmt.Sprintf("%s:%d", serMsg.ToAddress.Host, serMsg.ToAddress.Port)
		return NewMessage(serMsg.Channel, fromAddress, toAddress, serMsg.Text)
	}
	return nil, err
}

func (msg *Message) Id() string {
	return msg.id
}

func (msg *Message) Text() string {
	return msg.text
}

func (msg *Message) Channel() string {
	return msg.channel
}

func (msg *Message) FromAddress() *MessageAddress {
	return &msg.fromAddress
}

func (msg *Message) ToAddress() *MessageAddress {
	return &msg.toAddress
}

// serialise message
func (msg *Message) Serialise() (string, error) {
	msgToSerialise := serialisedMsg{
		msg.id,
		msg.text,
		msg.channel,
		msg.fromAddress,
		msg.toAddress,
	}
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(msgToSerialise)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

func v5UUID(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data))
}
func getHostAndPortFromAddress(address string) (string, int, error) {
	hostStr, portStr, hostPortErr := net.SplitHostPort(address)
	if hostPortErr != nil {
		return "", 0, hostPortErr
	}
	port, convErr := strconv.Atoi(portStr)
	if convErr != nil {
		return "", 0, convErr
	}
	return hostStr, port, nil
}
