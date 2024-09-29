package main

import (
	"fmt"
	"message"
)

func main() {
	fmt.Println("start")
	msg := message.New()
	msgText, err := msg.ToString()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(msgText)
	}
	fmt.Println("stop")
}
