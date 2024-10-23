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
	expectedMsg := createMessage(test, "Hello World", true)
	msgAtt1 := NewMsgAtt(HTTP_SERVER_REPONSE)
	msgAtt2 := NewMsgAtt(HTTP_SERVER_REQUEST)
	msgSub := NewMsgSub(HTTP_SERVER_SUBCRIPTION, msgAtt1, msgAtt2)
	err := ch.Publish(msgSub, expectedMsg)
	if err == nil {
		var receivedMsg Msg
		receivedMsg, err = ch.Subscribe(msgSub)
		if err == nil {
			if receivedMsg == expectedMsg {
				test.Fail()
			}
			if receivedMsg.GetId() != expectedMsg.GetId() {
				test.Fail()
			}
		}
	} else {
		test.Fatal(err)
	}
	ch.Close()
	runtime.GC()
	time.Sleep(5 * time.Second)
}
