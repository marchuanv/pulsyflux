package httpcontainers

import (
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"pulsyflux/util"
)

type httpRequestHandler struct {
	envelopes *sliceext.List[contracts.Envelope]
	rcvNvlp   chan contracts.Envelope
}

func (rh *httpRequestHandler) GetEnvelopes() *sliceext.List[contracts.Envelope] {
	return rh.envelopes
}

func (rh *httpRequestHandler) SetEnvelopes(envelopes *sliceext.List[contracts.Envelope]) {
	rh.envelopes = envelopes
}

func (rh *httpRequestHandler) Receive(nvlpTypeId contracts.TypeId[contracts.Envelope]) contracts.Envelope {
	path := containers.Get[contracts.Envelope](nvlpTypeId).GetUrl().Path
	for nvlp := range rh.rcvNvlp {
		if nvlp.GetUrl().Path == path {
			return nvlp
		}
	}
	return nil
}

func (rh *httpRequestHandler) Send(nvlpTypeId contracts.TypeId[contracts.Envelope]) {
}

func (rh *httpRequestHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	for _, nvlp := range rh.GetEnvelopes().All() {
		if request.URL.Path == nvlp.GetUrl().Path {
			nvlp.SetMsg(util.StringFromReader(request.Body))
			rh.rcvNvlp <- nvlp
		}
	}
	response.WriteHeader(http.StatusOK)
}
