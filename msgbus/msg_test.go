package msgbus

import (
	"testing"
)

func TestMsgEquality(test *testing.T) {
	msg1 := NewMessage("Hello World")
	msg2 := NewMessage("Hello World")
	msg3 := NewMessage("Hello John")
	if msg1 == msg2 {
		test.Fatal("CtorError: did not expect message pointers to be the same")
	}
	if msg3 == msg1 {
		test.Fatal("CtorError: expected message pointers to NOT be the same")
	}
	if msg3 == msg2 {
		test.Fatal("CtorError: expected message pointers to NOT be the same")
	}
}
func TestMsgSerialiseAndDeserialise(test *testing.T) {
	msg := NewMessage("Hello World")
	serialisedMsg := msg.Serialise()
	deserialisedMsg := NewDeserialisedMessage(serialisedMsg)
	if msg.GetId() != deserialisedMsg.GetId() {
		test.Fatal("expected deserialised message Id to equal original message Id")
	}
	if msg.String() != deserialisedMsg.String() {
		test.Fatal("expected deserialised message text to equal original message text")
	}
}
