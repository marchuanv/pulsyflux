package msgbus

import (
	"reflect"
	"testing"
)

func TestCreateMsgSerialiseAndDeserialise(test *testing.T) {
	origMsg := NewMessage("Hello World")
	msgStr := origMsg.Serialise()
	msg := NewDeserialisedMessage(msgStr)
	if !reflect.DeepEqual(origMsg, msg) {
		test.Fail()
	}
}
