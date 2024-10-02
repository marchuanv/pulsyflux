package main

import (
	"fmt"
	"pulsyflux/message"
)

func main() {
	fmt.Println("start")
	msg, err := message.New("Hello Bob")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(msg.Text)
	}
	fmt.Println("stop")
}
