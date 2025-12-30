package httpcontainer

import (
	"context"
	"pulsyflux/contracts"
	"testing"
	"time"

	"github.com/google/uuid"
)

func SetupTest(test *testing.T, uuidStr string) (contracts.HttpResHandler, contracts.HttpServer, contracts.HttpReq, uuid.UUID) {
	msgId, err := uuid.Parse(uuidStr)
	if err != nil {
		test.Fatal(err)
	}
	httpReq := InitialiseHttpReq(10 * time.Second)
	resHdl := InitialiseHttpResHandler(msgId, 200)
	server := InitialiseHttpServer("http", "localhost", 3000, "")
	resHdlMatch := server.GetResponseHandler(resHdl.MsgId())
	if resHdl != resHdlMatch {
		test.Fail()
	}
	return resHdl, server, httpReq, msgId
}

func TestHttpServerSuccess(test *testing.T) {
	uuidStr := uuid.NewString()
	handler, server, req, msgId := SetupTest(test, uuidStr)
	defer server.Stop()
	expectedMsg := `{"message":"success-test","msg_id":"` + uuidStr + `"}`
	go server.Start()
	go req.Send(server.GetAddress(), msgId, expectedMsg)
	rcvMsg, received := handler.ReceiveRequest(context.Background())
	if !received {
		test.Fail()
	}
	if rcvMsg != contracts.Msg(expectedMsg) {
		test.Fail()
	}
	handler.RespondToRequest(context.Background(), contracts.Msg(expectedMsg))
}

func TestHttpServerTimeout(test *testing.T) {

	uuidStr := uuid.NewString()
	handler, server, req, msgId := SetupTest(test, uuidStr)
	expectedMsg := `{"message":"timeout-test","msg_id":"` + uuidStr + `"}`

	go server.Start()

	defer func() {
		time.Sleep(15 * time.Second)
		server.Stop()
	}()

	go func() {
		handler.ReceiveRequest(context.Background())
		time.Sleep(10 * time.Second)
		handler.RespondToRequest(context.Background(), contracts.Msg(expectedMsg))
	}()

	go func() {
		responseStr := req.Send(server.GetAddress(), msgId, expectedMsg)
		if responseStr != expectedMsg {
			test.Fail()
		}
	}()
}
