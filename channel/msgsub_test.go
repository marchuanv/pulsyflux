package channel

import (
	"testing"
)

const (
	HTTP_SERVER_SUBCRIPTION MsgSubId = "33f06644-cc47-4b26-87ad-f089cef38fd1"
	HTTP_SERVER_REPONSE     MsgAttId = "ca1d8b65-36e4-4eb0-a7c4-d1568beb369b"
	HTTP_SERVER_REQUEST     MsgAttId = "91088134-edcf-4f8d-9bbc-0d7c928a91b0"
)

func TestDecoratedMsgSub(test *testing.T) {

	msgAtt1 := NewMsgAtt(HTTP_SERVER_REPONSE)
	msgAtt2 := NewMsgAtt(HTTP_SERVER_REQUEST)
	msgSub := NewMsgSub(HTTP_SERVER_SUBCRIPTION, msgAtt1, msgAtt2)

	if msgSub.id == HTTP_SERVER_SUBCRIPTION {
		test.Fail()
	}
	if string(msgSub.id) == string(HTTP_SERVER_SUBCRIPTION) {
		test.Fail()
	}
	if string(msgSub.id) == string(HTTP_SERVER_REPONSE) {
		test.Fail()
	}
	if string(msgSub.id) == string(HTTP_SERVER_REPONSE) {
		test.Fail()
	}
}
