package httpcontainers

import (
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
)

type httpResponse struct {
	nvlpId            contracts.TypeId[contracts.Envelope]
	nvlpInc           *chan contracts.TypeId[contracts.Envelope]
	nvlpOut           *chan contracts.TypeId[contracts.Envelope]
	successStatusCode *int
	successStatusMsg  *string
}

func (r *httpResponse) SetNvlpId(nvlpId *contracts.TypeId[contracts.Envelope]) {
	r.nvlpId = *nvlpId
}

func (r *httpResponse) SetNvlpInc(nvlpInc *chan contracts.TypeId[contracts.Envelope]) {
	if r.nvlpInc == nil {
		r.nvlpInc = nvlpInc // use the pointer passed in
		return
	}
	*(r.nvlpInc) = *nvlpInc
}

func (r *httpResponse) SetNvlpOut(nvlpOut *chan contracts.TypeId[contracts.Envelope]) {
	if r.nvlpOut == nil {
		r.nvlpOut = nvlpOut // use the pointer passed in
		return
	}
	*(r.nvlpOut) = *nvlpOut
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

func (r *httpResponse) GetMsg() any {
	<-*r.nvlpInc
	nvlp := containers.Get[contracts.Envelope](r.nvlpId)
	return nvlp.GetMsg()
}

func (r *httpResponse) SetMsg(msg any) {
	nvlp := containers.Get[contracts.Envelope](r.nvlpId)
	_msg := msg.(string)
	nvlp.SetMsg(&_msg)
	*r.nvlpOut <- r.nvlpId
}

func (r *httpResponse) Handle(reqHeader http.Header, reqBody string) (reason string, statusCode int, resBody string) {
	envelope_Id := reqHeader.Get("envelope_id")
	resBody = ""
	if envelope_Id == "" {
		reason = "missing envelope_id header"
		statusCode = http.StatusBadRequest
		return
	}

	if envelope_Id != string(r.nvlpId) {
		reason = "no handler for envelope id " + envelope_Id
		statusCode = http.StatusBadRequest
		return
	}
	nvlp := containers.Get[contracts.Envelope](r.nvlpId)
	nvlp.SetMsg(&reqBody)
	*r.nvlpInc <- r.nvlpId
	<-*r.nvlpOut
	reason = "success"
	statusCode = http.StatusOK
	resBody = *nvlp.GetMsg()
	return
}
