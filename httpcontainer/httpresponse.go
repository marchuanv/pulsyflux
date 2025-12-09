package httpcontainer

import (
	"net/http"
	"pulsyflux/contracts"
)

type httpResponse struct {
	msgTypeId         *contracts.TypeId[contracts.Msg]
	incMsg            *chan contracts.Msg
	outMsg            *chan contracts.Msg
	successStatusCode *int
	successStatusMsg  *string
}

func (r *httpResponse) SetMsgTypeId(msgTypeId *contracts.TypeId[contracts.Msg]) {
	if r.msgTypeId == nil {
		r.msgTypeId = msgTypeId // use the pointer passed in
		return
	}
	*(r.msgTypeId) = *msgTypeId
}

func (r *httpResponse) SetIncMsg(incMsg *chan contracts.Msg) {
	if r.incMsg == nil {
		r.incMsg = incMsg // use the pointer passed in
		return
	}
	*(r.incMsg) = *incMsg
}

func (r *httpResponse) SetOutMsg(outMsg *chan contracts.Msg) {
	if r.outMsg == nil {
		r.outMsg = outMsg // use the pointer passed in
		return
	}
	*(r.outMsg) = *outMsg
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
		reason = "missing envelope_id header"
		statusCode = http.StatusBadRequest
		return
	}

	if message_id != string(*r.msgTypeId) {
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
