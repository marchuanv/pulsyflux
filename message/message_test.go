package message

import (
	"testing"
)

func testMsgCtorWithoutArgs(msgText string, t *testing.T) *Message {
	msgPtr, err := New(msgText)
	if err != nil {
		t.Errorf("CtorError: failed to create a message with errors")
		t.Logf("%s", err)
	}
	return msgPtr
}

func testUUIDEquality(t *testing.T) {
	msgRef1 := testMsgCtorWithoutArgs("Hello World", t)
	msgRef2 := testMsgCtorWithoutArgs("Hello World", t)
	msgRef3 := testMsgCtorWithoutArgs("Hello John", t)
	if msgRef1 != msgRef2 {
		t.Errorf("CtorError: expected message pointers to be the same")
	}
	if msgRef1 == msgRef3 {
		t.Errorf("CtorError: expected message pointers to NOT be the same")
	}
}
func TestMsgCtor(t *testing.T) {
	testUUIDEquality(t)
}
