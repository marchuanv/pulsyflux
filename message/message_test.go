package message

import (
	"testing"
)

func createMessage(test *testing.T, channel string, fromAddress string, toAddress string, text string, exitOnError bool) *Message {
	msgPtr, err := NewMessage(channel, fromAddress, toAddress, text)
	if err == nil {
		return msgPtr
	}
	if exitOnError {
		test.Fatalf("failed to create a new message, exiting test...") // Exits the program
	}
	return nil
}

func createDeserialisedMessage(test *testing.T, serialisedMsg string, exitOnError bool) *Message {
	message, err := NewDeserialiseMessage(serialisedMsg)
	if err == nil {
		return message
	}
	if exitOnError {
		test.Fatalf("failed to deserialise message, exiting test...") // Exits the program
	}
	return nil
}

func serialiseMessage(test *testing.T, message *Message, exitOnError bool) string {
	serialisedMsg, err := message.Serialise()
	if err == nil {
		return serialisedMsg
	}
	if exitOnError {
		test.Fatalf("failed to serialise message, exiting test...") // Exits the program
	}
	return ""
}

func TestCreatingMessageWithBadAddressVariationA(test *testing.T) {
	badMsg := createMessage(test, "channel", "localhost:", ":3000", "Hello World", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestCreatingMessageWithBadAddressVariationB(test *testing.T) {
	badMsg := createMessage(test, "channel", "", "localhost:3000", "Hello World", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestCreatingMessageWithBadText(test *testing.T) {
	badMsg := createMessage(test, "channel", "localhost:3000", "localhost:3000", "", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestCreatingMessageWithBadChannel(test *testing.T) {
	badMsg := createMessage(test, "", "localhost:3000", "localhost:3000", "Hello World", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestMsgEquality(test *testing.T) {
	msgRef1 := createMessage(test, "channel", "localhost:3000", "localhost:3000", "Hello World", true)
	msgRef2 := createMessage(test, "channel", "localhost:3000", "localhost:3000", "Hello World", true)
	msgRef3 := createMessage(test, "channel", "localhost:3000", "localhost:3000", "Hello John", true)
	if msgRef1 != msgRef2 {
		test.Fatal("CtorError: expected message pointers to be the same")
	}
	if msgRef3 == msgRef1 {
		test.Fatal("CtorError: expected message pointers to NOT be the same")
	}
	if msgRef3 == msgRef2 {
		test.Fatal("CtorError: expected message pointers to NOT be the same")
	}
}
func TestMsgSerialiseAndDeserialise(test *testing.T) {
	msg := createMessage(test, "channel", "localhost:3000", "localhost:3000", "Hello World", true)
	serialisedMsg := serialiseMessage(test, msg, true)
	deserialisedMsg := createDeserialisedMessage(test, serialisedMsg, true)
	if msg == deserialisedMsg {
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
