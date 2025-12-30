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
	httpReq := InitialiseHttpReq()
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
	server.Start()
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

func TestHttpClientTimeout(test *testing.T) {

	uuidStr := uuid.NewString()
	handler, server, req, msgId := SetupTest(test, uuidStr)

	defer func() {
		server.Stop()
		err := recover()
		if err == nil || err != "client request timed out" {
			test.Log(err)
			test.Fail()
		}
	}()

	expectedMsg := `{"message":"timeout-test","msg_id":"` + uuidStr + `"}`

	server.Start()

	go func() {
		handler.ReceiveRequest(context.Background())
		time.Sleep(3 * time.Second) //similate processing delay
		handler.RespondToRequest(context.Background(), contracts.Msg(expectedMsg))
	}()

	req.Send(server.GetAddress(), msgId, expectedMsg)
}

func TestHttpServerTimeout(test *testing.T) {

	uuidStr := uuid.NewString()
	handler, server, req, msgId := SetupTest(test, uuidStr)

	defer func() {
		server.Stop()
		err := recover()
		if err != nil && err != "server closed the connection unexpectedly (EOF)" {
			test.Log(err)
			test.Fail()
		}
	}()

	expectedMsg := `{"message":"timeout-test","msg_id":"` + uuidStr + `"}`

	server.Start()

	go func() {
		handler.ReceiveRequest(context.Background())
		time.Sleep(26 * time.Second) //similate processing delay
		handler.RespondToRequest(context.Background(), contracts.Msg(expectedMsg))
	}()

	req.Send(server.GetAddress(), msgId, expectedMsg)
}
