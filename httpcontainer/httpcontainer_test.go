package httpcontainer

import (
	"testing"

	"github.com/google/uuid"
)

func TestHttpServer(test *testing.T) {
	msgId := uuid.New()
	status := 200
	myHttpResHandler := InitialiseHttpResHandler(msgId, status)
	server := InitialiseHttpServer("http", "localhost", 3000, "")
	resHdlMatch := server.Response(myHttpResHandler.MsgId())
	if myHttpResHandler != resHdlMatch {
		test.Fail()
	}
	// server.Start()
	// test.Fail()
	// test.Log(server)
}
