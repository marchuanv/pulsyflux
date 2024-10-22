package channel

import (
	"testing"

	"github.com/google/uuid"
)

func createChannel(test *testing.T, id uuid.UUID, exitOnError bool) *Channel {
	ch, err := Open(id)
	if err == nil {
		return ch
	}
	if exitOnError {
		test.Fatal(err) // Exits the program
	}
	return ch
}

func TestCreateChannel(test *testing.T) {
	channelId := uuid.New()
	ch := createChannel(test, channelId, true)
	msgId := uuid.New()
	expectedMsg := createMessage(test, msgId, "localhost:3000", "localhost:4000", "Hello World", true)
	ch.Push(expectedMsg)
	receiMsgs, err := ch.Pop()
	if err == nil {
		if receiMsgs[0] != expectedMsg {
			test.Fail()
		}
	} else {
		test.Fatal(err)
	}
}
