package msgbus

import (
	"bytes"
	"net/http"
	"testing"
)

func getMsgBus(test *testing.T, address string, channel string) *MsgBus {
	msgBus, err := Get(address, channel)
	if err != nil {
		test.Fatalf("exiting test") // Exits the program
		return nil
	} else {
		return msgBus
	}
}

func TestShouldProvideSameMessageBusForSameAddress(test *testing.T) {
	msgBus1 := getMsgBus(test, "localhost:4000", "2f7544de-9830-4fff-8ac8-a6f56c14ee2b")
	if msgBus1 == nil {
		test.Fail()
		return
	}
	msgBus2 := getMsgBus(test, "localhost:4000", "4debab87-eb33-4b3b-b135-572c7c0947b1")
	if msgBus2 == nil {
		test.Fail()
		return
	}
	if msgBus1 != msgBus2 {
		test.Fail()
	}
	jsonBody := []byte("hello, server")
	bodyReader := bytes.NewReader(jsonBody)
	http.Post("http://localhost:4000", "text/plain", bodyReader)
}
