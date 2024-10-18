package msgbus

import (
	"encoding/base64"
	"fmt"
	"pulsyflux/connect"
	"pulsyflux/message"
	"pulsyflux/msgq"
	"pulsyflux/util"
)

type MsgBus struct {
	queue *msgq.MsgQueue
	conn  *connect.Connection
}

var msgBuses map[string]*MsgBus

func Get(address string, channel string) (*MsgBus, error) {
	_, _, err := util.GetHostAndPortFromAddress(address)
	if err != nil {
		return nil, err
	}
	if msgBuses == nil {
		msgBuses = make(map[string]*MsgBus)
	}
	msgBus, exists := msgBuses[address]
	if exists {
		return msgBus, nil
	}
	msgBus = &MsgBus{}
	msgBuses[address] = msgBus
	queue, err := msgq.Get(channel)
	if err != nil {
		return nil, err
	}
	msgBus.queue = queue
	conn, err := connect.New(address)
	if err != nil {
		return nil, err
	}
	msgBus.conn = conn
	return msgBus, nil
}
func (msgBus *MsgBus) Stop() {
	msgBus.conn.Close()
}
func (msgBus *MsgBus) Start() {
	go (func() {
		channel := msgBus.conn.Channel()
		for base64Msg := range channel {
			bytes, err := base64.StdEncoding.DecodeString(base64Msg)
			if err == nil {
				msgStr := fmt.Sprint(bytes)
				msg, err := message.NewDeserialiseMessage(msgStr)
				if err == nil {
					msgBus.queue.Enqueue(&msg)
				}
			}
		}
	})()
}
