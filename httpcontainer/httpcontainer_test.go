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

func TestMultipleHttpServerAsyncStart(test *testing.T) {
	_, server1, _, _ := SetupTest(test, uuid.NewString())
	_, server2, _, _ := SetupTest(test, uuid.NewString())
	defer func() {
		server1.Stop()
		server2.Stop()
	}()
	server1.Start()
	server2.Start()
	time.Sleep(10 * time.Second)
}

func TestSameServerMultipleStart(test *testing.T) {
	_, server, _, _ := SetupTest(test, uuid.NewString())
	defer server.Stop()
	server.Start()
	server.Start()
	time.Sleep(10 * time.Second)
}

func TestHttpServerSuccess(test *testing.T) {

	uuidStr := uuid.NewString()
	handler, server, req, msgId := SetupTest(test, uuidStr)

	defer server.Stop()
	server.Start()

	expectedMsg := `{"message":"success-test","msg_id":"` + uuidStr + `"}`
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
	server.Start()

	expectedMsg := `{"message":"timeout-test","msg_id":"` + uuidStr + `"}`

	go func() {
		handler.ReceiveRequest(context.Background())
		time.Sleep(26 * time.Second) //similate processing delay
		handler.RespondToRequest(context.Background(), contracts.Msg(expectedMsg))
	}()

	status, _ := req.Send(server.GetAddress(), msgId, expectedMsg)
	if status.Code() != 200 {
		test.Fail()
	}
}

func TestHttpServerTimeout(test *testing.T) {

	uuidStr := uuid.NewString()
	handler, server, _, msgId := SetupTest(test, uuidStr)

	req := newHttpReq(
		&timeDuration{60 * time.Second},
		&timeDuration{0},
		&timeDuration{60 * time.Second},
	)

	defer func() {
		server.Stop()
		err := recover()
		if err != nil && err != "server closed the connection unexpectedly (EOF)" {
			test.Log(err)
			test.Fail()
		}
	}()

	server.Start()

	expectedMsg := `{"message":"timeout-test","msg_id":"` + uuidStr + `"}`

	go func() {
		handler.ReceiveRequest(context.Background())
		time.Sleep(26 * time.Second) //similate processing delay
		handler.RespondToRequest(context.Background(), contracts.Msg(expectedMsg))
	}()

	req.Send(server.GetAddress(), msgId, expectedMsg)
}
