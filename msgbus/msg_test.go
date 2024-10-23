package msgbus

import (
	"testing"
)

func createMessage(test *testing.T, text string, exitOnError bool) Msg {
	msg, err := NewMessage(text)
	if err != nil {
		if exitOnError {
			test.Fatalf("failed to create a new message, exiting test...") // Exits the program
		} else {
			test.Log(err)
		}
	}
	return msg
}

func createDeserialisedMessage(test *testing.T, serialisedMsg string, exitOnError bool) Msg {
	msg, err := NewDeserialisedMessage(serialisedMsg)
	if err != nil {
		if exitOnError {
			test.Fatalf("failed to create a new message, exiting test...") // Exits the program
		} else {
			test.Error(err)
		}
	}
	return msg
}

func serialiseMessage(test *testing.T, msg Msg, exitOnError bool) string {
	serialisedMsg, err := msg.Serialise()
	if err == nil {
		return serialisedMsg
	}
	if exitOnError {
		test.Fatalf("failed to serialise message, exiting test...") // Exits the program
	}
	return ""
}

func TestCreatingMessageWithBadText(test *testing.T) {
	badMsg := createMessage(test, "", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestMsgEquality(test *testing.T) {
	msgRef1 := createMessage(test, "Hello World", true)
	msgRef2 := createMessage(test, "Hello World", true)
	msgRef3 := createMessage(test, "Hello John", true)
	if msgRef1 == msgRef2 {
		test.Fatal("CtorError: did not expect message pointers to be the same")
	}
	if msgRef3 == msgRef1 {
		test.Fatal("CtorError: expected message pointers to NOT be the same")
	}
	if msgRef3 == msgRef2 {
		test.Fatal("CtorError: expected message pointers to NOT be the same")
	}
}
func TestMsgSerialiseAndDeserialise(test *testing.T) {
	msg := createMessage(test, "Hello World", true)
	serialisedMsg := serialiseMessage(test, msg, true)
	deserialisedMsg := createDeserialisedMessage(test, serialisedMsg, true)
	if msg.GetId() != deserialisedMsg.GetId() {
		test.Fatal("expected deserialised message Id to equal original message Id")
	}
	if msg.String() != deserialisedMsg.String() {
		test.Fatal("expected deserialised message text to equal original message text")
	}
}
