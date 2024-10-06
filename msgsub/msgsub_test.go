package msgsub

import (
	"pulsyflux/message"
	"testing"
)

func newMessage(test *testing.T, channel string, fromAddress string, toAddress string, text string) *message.Message {
	msg, err := message.NewMessage(channel, fromAddress, toAddress, text)
	if err == nil {
		return msg
	}
	test.Fatalf("exiting test") // Exits the program
	return nil
}
func Something(t *testing.T) {
}
