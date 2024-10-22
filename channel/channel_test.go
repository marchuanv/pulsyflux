package channel

import (
	"runtime"
	"testing"
	"time"

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
	attributes := []string{"Valid"}
	expectedMsg := createMessage(test, attributes, "Hello World", true)
	ch.Push(expectedMsg)
	receiMsgs, err := ch.Pop()
	if err == nil {
		if receiMsgs[0] == expectedMsg {
			test.Fail()
		}
		if receiMsgs[0].GetId() != expectedMsg.GetId() {
			test.Fail()
		}
	} else {
		test.Fatal(err)
	}
	ch.Close()
	runtime.GC()
	time.Sleep(5 * time.Second)
}
