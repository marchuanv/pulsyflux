package httpcontainer

import (
	"context"
	"fmt"
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"strings"

	"github.com/google/uuid"
)

type status struct {
	incMsg            chan contracts.Msg
	outMsg            chan contracts.Msg
	successStatusCode int
	successStatusMsg  string
	msgId             uuid.UUID
}

func (res *httpResHandler) Init(successStatusCode int, successStatusMsg string, msgId uuid.UUID) {
	res.incMsg = make(chan contracts.Msg, 1)
	res.outMsg = make(chan contracts.Msg, 1)
	res.successStatusCode = successStatusCode
	res.successStatusMsg = successStatusMsg
	res.msgId = msgId
}

func (res *httpResHandler) ReceiveRequestMsg(
	ctx context.Context,
) (contracts.Msg, bool) {
	select {
	case msg := <-res.incMsg:
		return msg, true
	case <-ctx.Done():
		return "", false
	}
}

func (res *httpResHandler) SendResponseMsg(
	ctx context.Context,
	msg contracts.Msg,
) bool {
	select {
	case res.outMsg <- msg:
		return true
	case <-ctx.Done():
		return false
	}
}

func (res *httpResHandler) handle(
	ctx context.Context,
	reqHeader http.Header,
	reqBody string,
) (reason string, statusCode int, resBody string) {

	if !res.containsMsgID(reqBody) {
		reason = fmt.Sprintf("message id %s is missing from request body", res.msgId.String())
		statusCode = http.StatusBadRequest
		return
	}

	//forward
	select {
	case res.incMsg <- contracts.Msg(reqBody):
	case <-ctx.Done():
		return "request cancelled", http.StatusRequestTimeout, ""
	}

	//wait for response
	select {
	case outMsg := <-res.outMsg:
		return res.successStatusMsg, res.successStatusCode, string(outMsg)
	case <-ctx.Done():
		return "processing timeout", http.StatusGatewayTimeout, ""
	}
}

func (h *httpResHandler) containsMsgID(body string) bool {
	return strings.Contains(body, h.msgId.String())
}

func NewHttpResponseContainer(successStatusCode int, successStatusMsg string, msgId uuid.UUID) contracts.Container4[httpResHandler, int, *int, string, *string, uuid.UUID, *uuid.UUID, *httpResHandler] {
	return containers.NewContainer4[httpResHandler, int, *int, string, *string, uuid.UUID, *uuid.UUID, *httpResHandler](successStatusCode, successStatusMsg, msgId)
}
