package httpcontainer

import (
	"context"
	"fmt"
	"net/http"
	"pulsyflux/contracts"
)

type httpResHandler struct {
	incMsg            chan contracts.Msg
	outMsg            chan contracts.Msg
	successStatusCode int
	successStatusMsg  string
	msgId             contracts.MsgId
}

// Receive request message (non-blocking)
func (res *httpResHandler) ReceiveRequest(ctx context.Context) (contracts.Msg, bool) {
	select {
	case msg := <-res.incMsg:
		return msg, true
	case <-ctx.Done():
		return "", false
	}
}

// Respond to request (non-blocking)
func (res *httpResHandler) RespondToRequest(ctx context.Context, msg contracts.Msg) bool {
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

// Core request handler
func (res *httpResHandler) handle(ctx context.Context, reqBody string) (reason string, statusCode int, resBody string) {
	if !res.containsMsgID(reqBody) {
		reason = fmt.Sprintf("message id %s is missing from request body", res.msgId.String())
		statusCode = http.StatusBadRequest
		return
	}

	_reqBody := reqBody[len(res.msgId.String()):] // trim MsgId prefix

	// Forward request
	select {
	case res.incMsg <- contracts.Msg(_reqBody):
	case <-ctx.Done():
		return "request cancelled", http.StatusRequestTimeout, ""
	}

	// Wait for response
	select {
	case outMsg := <-res.outMsg:
		return res.successStatusMsg, res.successStatusCode, string(outMsg)
	case <-ctx.Done():
		return "processing timeout", http.StatusGatewayTimeout, ""
	}
}

func (res *httpResHandler) containsMsgID(body string) bool {
	return len(body) >= len(res.msgId.String()) && body[:len(res.msgId.String())] == res.msgId.String()
}

// Factory function
func newHttpResHandler(httpStatus contracts.HttpStatus, msgId contracts.MsgId) contracts.HttpResHandler {
	response := &httpResHandler{
		incMsg:            make(chan contracts.Msg, 1),
		outMsg:            make(chan contracts.Msg, 1),
		successStatusCode: httpStatus.Code(),
		successStatusMsg:  httpStatus.String(),
		msgId:             msgId,
	}

	// Register in global map
	resOnce.Do(func() {
		responsesMap = make(map[contracts.MsgId]*httpResHandler)
	})
	mu.Lock()
	responsesMap[msgId] = response
	mu.Unlock()

	return response
}
