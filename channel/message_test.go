package channel

import (
	"testing"

	"github.com/google/uuid"
)

func createMessage(test *testing.T, id uuid.UUID, fromAddress string, toAddress string, text string, exitOnError bool) *Message {
	msg, err := New(id, fromAddress, toAddress, text)
	if err != nil {
		if exitOnError {
			test.Fatalf("failed to create a new message, exiting test...") // Exits the program
		} else {
			test.Log(err)
		}
	}
	return msg
}

func createDeserialisedMessage(test *testing.T, serialisedMsg string, exitOnError bool) *Message {
	msg, err := deserialise(serialisedMsg)
	if err != nil {
		if exitOnError {
			test.Fatalf("failed to create a new message, exiting test...") // Exits the program
		} else {
			test.Error(err)
		}
	}
	return msg
}

func serialiseMessage(test *testing.T, message *Message, exitOnError bool) string {
	serialisedMsg, err := message.serialise()
	if err == nil {
		return serialisedMsg
	}
	if exitOnError {
		test.Fatalf("failed to serialise message, exiting test...") // Exits the program
	}
	return ""
}

func TestCreatingMessageWithBadAddressVariationA(test *testing.T) {
	msgId := uuid.New()
	badMsg := createMessage(test, msgId, "localhost:", ":3000", "Hello World", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestCreatingMessageWithBadAddressVariationB(test *testing.T) {
	msgId := uuid.New()
	badMsg := createMessage(test, msgId, "", "localhost:3000", "Hello World", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestCreatingMessageWithBadText(test *testing.T) {
	msgId := uuid.New()
	badMsg := createMessage(test, msgId, "localhost:3000", "localhost:3000", "", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestMsgEquality(test *testing.T) {
	msgId := uuid.New()
	msgRef1 := createMessage(test, msgId, "localhost:3000", "localhost:3000", "Hello World", true)
	msgRef2 := createMessage(test, msgId, "localhost:3000", "localhost:3000", "Hello World", true)
	msgRef3 := createMessage(test, msgId, "localhost:3000", "localhost:3000", "Hello John", true)
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
	msgId := uuid.New()
	msg := createMessage(test, msgId, "localhost:3000", "localhost:3000", "Hello World", true)
	serialisedMsg := serialiseMessage(test, msg, true)
	deserialisedMsg := createDeserialisedMessage(test, serialisedMsg, true)
	if msg.id == deserialisedMsg.id {
		if msg.FromAddress() == deserialisedMsg.FromAddress() {
		} else {
			test.Fatal("expected deserialised message text to equal original message text")
		}
		if msg.ToAddress() == deserialisedMsg.ToAddress() {
		} else {
			test.Fatal("expected deserialised message text to equal original message text")
		}
	} else {
		test.Fatal("expected deserialised message Id to equal original message Id")
	}
}
