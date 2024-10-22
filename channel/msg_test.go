package channel

import (
	"testing"
)

func createMessage(test *testing.T, attributeNames []string, text string, exitOnError bool) Msg {
	attributes := []MsgAtt{}
	for _, attName := range attributeNames {
		att, err := NewMsgAtt(attName)
		attributes = append(attributes, att)
		if err == nil {
		} else {
			if exitOnError {
				test.Fatalf("failed to create a new message, exiting test...") // Exits the program
			} else {
				test.Log(err)
			}
		}
	}
	msg, err := NewMessage(attributes, text)
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

func TestCreatingMessageWithNoAttributes(test *testing.T) {
	attributes := []string{}
	badMsg := createMessage(test, attributes, "Hello World", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestCreatingMessageWithBadText(test *testing.T) {
	attributes := []string{"BadText"}
	badMsg := createMessage(test, attributes, "", false)
	if badMsg != nil {
		test.Fatalf("expected test to fail")
	}
}
func TestMsgEquality(test *testing.T) {
	attributes := []string{"Valid"}
	msgRef1 := createMessage(test, attributes, "Hello World", true)
	msgRef2 := createMessage(test, attributes, "Hello World", true)
	msgRef3 := createMessage(test, attributes, "Hello John", true)
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
	attributes := []string{"Valid"}
	msg := createMessage(test, attributes, "Hello World", true)
	serialisedMsg := serialiseMessage(test, msg, true)
	deserialisedMsg := createDeserialisedMessage(test, serialisedMsg, true)
	if msg.GetId() != deserialisedMsg.GetId() {
		test.Fatal("expected deserialised message Id to equal original message Id")
	}
}
