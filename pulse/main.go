package main

import (
	"fmt"
	"pulsyflux/msgbus"
	"pulsyflux/subscriptions"
)

func main() {

	var stopMsgBusCh *msgbus.Channel
	var startMsgBusCh *msgbus.Channel
	var err error

	startMsgBusCh, err = msgbus.New(subscriptions.START_MESSAGE_BUS)
	if err != nil {
		panic(err)
	}
	err = startMsgBusCh.Open()
	if err != nil {
		panic(err)
	}
	stopMsgBusCh, err = msgbus.New(subscriptions.STOP_MESSAGE_BUS)
	if err != nil {
		panic(err)
	}
	err = stopMsgBusCh.Open()
	if err != nil {
		panic(err)
	}

	msg, err := msgbus.NewMessage("localhost:3000")
	if err != nil {
		panic(err)
	}

	err = startMsgBusCh.Publish(msg)
	if err != nil {
		panic(err)
	}
	msg, err = stopMsgBusCh.Subscribe()
	if err != nil {
		panic(err)
	}

	fmt.Print(msg.Serialise())

	fmt.Println("stop")

}
