package httpcontainer

import (
	"context"
	"fmt"
	"net/http"
	"pulsyflux/shared"
)

type httpResHandler struct {
	incMsg            chan shared.Msg
	outMsg            chan shared.Msg
	successStatusCode int
	successStatusMsg  string
	msgId             shared.MsgId
}

// Receive request message (non-blocking)
func (res *httpResHandler) ReceiveRequest(ctx context.Context) (shared.Msg, bool) {
	select {
	case msg := <-res.incMsg:
		return msg, true
	case <-ctx.Done():
		return "", false
	}
}

// Respond to request (non-blocking)
func (res *httpResHandler) RespondToRequest(ctx context.Context, msg shared.Msg) bool {
	select {
	case res.outMsg <- msg:
		return true
	case <-ctx.Done():
		return false
	}
}

func (res *httpResHandler) MsgId() shared.MsgId {
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
	case res.incMsg <- shared.Msg(_reqBody):
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
func newHttpResHandler(httpStatus shared.HttpStatus, msgId shared.MsgId) shared.HttpResHandler {
	response := &httpResHandler{
		incMsg:            make(chan shared.Msg, 1),
		outMsg:            make(chan shared.Msg, 1),
		successStatusCode: httpStatus.Code(),
		successStatusMsg:  httpStatus.String(),
		msgId:             msgId,
	}

	// Register in global map
	resOnce.Do(func() {
		responsesMap = make(map[shared.MsgId]*httpResHandler)
	})
	mu.Lock()
	responsesMap[msgId] = response
	mu.Unlock()

	return response
}
