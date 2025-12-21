package httpcontainer

import (
	"context"
	"fmt"
	"net/http"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"strings"
)

type httpResHandler struct {
	incMsg            chan contracts.Msg
	outMsg            chan contracts.Msg
	successStatusCode int
	successStatusMsg  string
	msgId             contracts.MsgId
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

func (res *httpResHandler) MsgId() contracts.MsgId {
	return res.msgId
}

func (res *httpResHandler) handle(
	ctx context.Context,
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

func newHttpResHandler(httpStatus contracts.HttpStatus, msgId contracts.MsgId) contracts.HttpResHandler {
	response := &httpResHandler{
		make(chan contracts.Msg, 1),
		make(chan contracts.Msg, 1),
		httpStatus.Code(),
		httpStatus.String(),
		msgId,
	}
	resOnce.Do(func() {
		responses = sliceext.NewList[*httpResHandler]()
	})
	responses.Add(response)
	return response
}
