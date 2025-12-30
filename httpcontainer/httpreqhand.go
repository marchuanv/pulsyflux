package httpcontainer

import (
	"net/http"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"pulsyflux/util"
	"sync"
)

var (
	responses *sliceext.List[*httpResHandler]
	resOnce   sync.Once
)

type httpReqHandler struct {
}

func (rh *httpReqHandler) getResHandler(msgId contracts.MsgId) contracts.HttpResHandler {
	for _, res := range responses.All() {
		if res.msgId == msgId {
			return res
		}
	}
	return nil
}

func (rh *httpReqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // Use the request context
	if responses.Len() == 0 {
		http.Error(w, "no http responses configured", http.StatusInternalServerError)
		return
	}
	reqBody := util.StringFromReader(r.Body)
	var reason string
	var statusCode int
	var resBody string
	for _, res := range responses.All() {

		// CRITICAL: Check if the server is shutting down BEFORE the next iteration
		select {
		case <-ctx.Done():
			return // Exit immediately so the server knows this request is done
		default:
		}

		reason, statusCode, resBody = res.handle(ctx, reqBody)
		if statusCode == res.successStatusCode {
			w.WriteHeader(statusCode)
			w.Write([]byte(resBody))
			return
		}
	}
	http.Error(w, reason, statusCode)
}

func newHttpReqHandler() contracts.HttpReqHandler {
	resOnce.Do(func() {
		responses = sliceext.NewList[*httpResHandler]()
	})
	return &httpReqHandler{}
}
