package httpcontainer

import (
	"net/http"
	"pulsyflux/shared"
	"pulsyflux/util"
	"sync"
)

var (
	responsesMap map[shared.MsgId]*httpResHandler
	resOnce      sync.Once
	mu           sync.RWMutex // protects responsesMap
)

type httpReqHandler struct{}

func (rh *httpReqHandler) getResHandler(msgId shared.MsgId) shared.HttpResHandler {
	mu.RLock()
	defer mu.RUnlock()
	if res, exists := responsesMap[msgId]; exists {
		return res
	}
	return nil
}

func (rh *httpReqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqBody, err := util.StringFromReader(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	// Find matching handler
	var targetRes *httpResHandler
	mu.RLock()
	for _, res := range responsesMap {
		if res.containsMsgID(reqBody) {
			targetRes = res
			break
		}
	}
	mu.RUnlock()

	if targetRes == nil {
		http.Error(w, "no matching handler found", http.StatusBadRequest)
		return
	}

	// Handle the request using the internal handle() method
	reason, statusCode, resBody := targetRes.handle(ctx, reqBody)

	if statusCode == targetRes.successStatusCode {
		w.WriteHeader(statusCode)
		w.Write([]byte(resBody))
		return
	}

	http.Error(w, reason, statusCode)
}

// Factory function
func newHttpReqHandler() shared.HttpReqHandler {
	resOnce.Do(func() {
		responsesMap = make(map[shared.MsgId]*httpResHandler)
	})
	return &httpReqHandler{}
}
