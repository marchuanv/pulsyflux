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

type message struct {
	Id          string
	Text        string
	Channel     string
	FromAddress MessageAddress
	ToAddress   MessageAddress
}

func New(channel string, fromAddress string, toAddress string, text string) (Message, error) {
	if len(channel) == 0 {
		return Message{}, errors.New("the channel argument is an empty string")
	}
	if !IsValidUUID(channel) {
		return Message{}, errors.New("the channel argument is not a uuid")
	}
	if len(fromAddress) == 0 {
		return Message{}, errors.New("the fromAddress argument is an empty string")
	}
	if len(toAddress) == 0 {
		return Message{}, errors.New("the toAddress argument is an empty string")
	}
	if len(text) == 0 {
		return Message{}, errors.New("the msgText argument is an empty string")
	}
	id := v5UUID(channel + fromAddress + toAddress)
	idStr := id.String()
	fromHost, fromPort, fromErr := getHostAndPortFromAddress(fromAddress)
	if fromErr != nil {
		return Message{}, fromErr
	}
	toHost, toPort, toErr := getHostAndPortFromAddress(toAddress)
	if toErr != nil {
		return Message{}, toErr
	}
	newMsg := Message{
		idStr,
		text,
		channel,
		MessageAddress{fromHost, fromPort}, //from
		MessageAddress{toHost, toPort},     //to
	}
	return newMsg, nil
}

func NewDeserialiseMessage(serialised string) (Message, error) {
	if len(serialised) == 0 {
		return Message{}, errors.New("the channel argument is an empty string")
	}
	msg := message{}
	by, err := base64.StdEncoding.DecodeString(serialised)
	if err != nil {
		return Message{}, err
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&msg)
	if err == nil {
		fromAddress := fmt.Sprintf("%s:%d", msg.FromAddress.Host, msg.FromAddress.Port)
		toAddress := fmt.Sprintf("%s:%d", msg.ToAddress.Host, msg.ToAddress.Port)
		msgChannel := msg.Channel
		msgText := msg.Text
		return New(msgChannel, fromAddress, toAddress, msgText)
	}
	return Message{}, err
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

func (msg *Message) FromAddress() MessageAddress {
	return msg.fromAddress
}

func (msg *Message) ToAddress() MessageAddress {
	return msg.toAddress
}

func (msg *Message) Dispose() {
}

// serialise message
func (msg *Message) Serialise() (string, error) {
	msgToSerialise := message{
		msg.id,
		msg.text,
		msg.channel,
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
func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
