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

type httpResHandler struct {
	incMsg            chan contracts.Msg
	outMsg            chan contracts.Msg
	successStatusCode int
	successStatusMsg  string
	msgId             uuid.UUID
}

func (res *httpResHandler) Init(
	con01 contracts.Container1[httpStatusCode, *httpStatusCode],
	con02 contracts.Container1[httpStatusMsg, *httpStatusMsg],
	con03 contracts.Container1[httpMsgId, *httpMsgId],
) {
	res.incMsg = make(chan contracts.Msg, 1)
	res.outMsg = make(chan contracts.Msg, 1)
	res.successStatusCode = int(con01.Get())
	res.successStatusMsg = string(con02.Get())
	res.msgId = uuid.UUID(con03.Get())
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

func NewHttpResponseContainer() contracts.Container4[
	httpResHandler,
	httpStatusCode,
	httpStatusMsg,
	httpMsgId,
	*httpStatusCode,
	*httpStatusMsg,
	*httpMsgId,
	*httpResHandler,
] {
	statusCode := NewHttpStatusContainer()
	statusMsg := NewHttpStatusMsgContainer()
	msgId := NewHttpMsgIdContainer()
	return containers.NewContainer4[
		httpResHandler,
		httpStatusCode,
		httpStatusMsg,
		httpMsgId,
		*httpStatusCode,
		*httpStatusMsg,
		*httpMsgId,
		*httpResHandler,
	](
		statusCode,
		statusMsg,
		msgId,
	)
}
