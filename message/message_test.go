package message

import (
	"testing"
)

func testMsgCtorWithArgs(msgText string, t *testing.T) (*Message, error) {
	msgPtr, err := New(msgText)
	if err != nil {
		t.Errorf("CtorError: failed to create a message with errors")
		t.Logf("%s", err)
		return nil, err
	} else {
		return msgPtr, nil
	}
}

func TestMsgEquality(t *testing.T) {
	msgRef1, err1 := testMsgCtorWithArgs("Hello World", t)
	msgRef2, err2 := testMsgCtorWithArgs("Hello World", t)
	msgRef3, err3 := testMsgCtorWithArgs("Hello John", t)
	if err1 == nil && err2 == nil {
		if msgRef1 != msgRef2 {
			t.Errorf("CtorError: expected message pointers to be the same")
		}
	}
	if err1 == nil && err3 == nil {
		if msgRef3 == msgRef1 {
			t.Errorf("CtorError: expected message pointers to NOT be the same")
		}
	}
	if err1 == nil && err2 == nil {
		if msgRef3 == msgRef2 {
			t.Errorf("CtorError: expected message pointers to NOT be the same")
		}
	}
}
