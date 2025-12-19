package httpcontainer

import (
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
)

type httpResponse struct {
	incMsg            *chan contracts.Msg
	outMsg            *chan contracts.Msg
	msgId             *contracts.MsgId[contracts.Msg]
	successStatusCode *int
	successStatusMsg  *string
}

func (rh *httpRequestHandler) Init() {

}

func (r *httpResponse) SetMsgId(msgId *contracts.MsgId[contracts.Msg]) {
	if r.msgId == nil {
		r.msgId = msgId // use the pointer passed in
		return
	}
	*(r.msgId) = *msgId
}

func (r *httpResponse) GetMsgId() contracts.MsgId[contracts.Msg] {
	return *r.msgId
}

func (r *httpResponse) GetSuccessStatusCode() *int {
	return r.successStatusCode
}

func (r *httpResponse) GetSuccessStatusMsg() *string {
	return r.successStatusMsg
}

func (r *httpResponse) SetSuccessStatusCode(code *int) {
	if r.successStatusCode == nil {
		r.successStatusCode = code // use the pointer passed in
		return
	}
	*(r.successStatusCode) = *code
}

func (r *httpResponse) SetSuccessStatusMsg(msg *string) {
	if r.successStatusMsg == nil {
		r.successStatusMsg = msg // use the pointer passed in
		return
	}
	*(r.successStatusMsg) = *msg
}

func (r *httpResponse) GetMsg() contracts.Msg {
	return <-*r.incMsg
}

func (r *httpResponse) SetMsg(msg contracts.Msg) {
	*r.outMsg <- msg
}

func (r *httpResponse) Handle(reqHeader http.Header, reqBody string) (reason string, statusCode int, resBody string) {
	message_id := reqHeader.Get("message_id")
	resBody = ""
	if message_id == "" {
		reason = "missing message_id in the header"
		statusCode = http.StatusBadRequest
		return
	}

	if message_id != string(*r.msgId) {
		reason = "no handler for message_id " + message_id
		statusCode = http.StatusBadRequest
		return
	}
	*r.incMsg <- contracts.Msg(reqBody)
	outMsg := <-*r.outMsg
	reason = "success"
	statusCode = http.StatusOK
	resBody = string(outMsg)
	return
}

func NewHttpResponseContainer() contracts.Container1[httpResponse, *httpResponse] {
	return containers.NewContainer1[httpResponse, *httpResponse]()
}
