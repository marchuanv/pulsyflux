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

var msgBuses = make(map[string]*MsgBus)

func Get(address string, channel string) (*MsgBus, error) {
	_, _, err := util.GetHostAndPortFromAddress(address)
	if err != nil {
		return nil, err
	}
	msgBusUUID := util.Newv5UUID(address + channel)
	msgBusId := msgBusUUID.String()
	msgBus, exists := msgBuses[msgBusId]
	if exists {
		return msgBus, nil
	}
	msgBus = &MsgBus{}
	msgBuses[msgBusId] = msgBus
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
func (msgBus *MsgBus) Queue() {
	go (func() {
		msgUtf8, err := msgBus.conn.GetMessage()
		if err != nil {
			fmt.Print(err)
		} else {
			bytes, err := base64.StdEncoding.DecodeString(msgUtf8)
			if err == nil {
				msgStr := string(bytes)
				msg, err := message.NewDeserialiseMessage(msgStr)
				if err == nil {
					msgBus.queue.Enqueue(&msg)
				}
			}
			msgBus.Queue()
		}
	})()
}
