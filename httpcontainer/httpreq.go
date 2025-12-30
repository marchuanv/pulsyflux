package httpcontainer

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"pulsyflux/contracts"
	"pulsyflux/util"
	"time"

	"github.com/google/uuid"
)

type httpReq struct {
	client *http.Client
}

func (r *httpReq) Send(addr contracts.URI, msgId uuid.UUID, content string) string {
	_content := util.ReaderFromString(content)
	resp, err := r.client.Post(addr.String(), "application/json", _content)
	if err != nil {

		if errors.Is(err, context.DeadlineExceeded) {
			panic("client request timed out")
		}

		if errors.Is(err, io.EOF) {
			panic("server closed the connection unexpectedly (EOF)")
		}

		var netErr net.Error
		errors.As(err, &netErr)
		if netErr.Timeout() {
			log.Println(netErr)
			panic(netErr)
		} else {
			panic(err)
		}
	}
	return util.StringFromReader(resp.Body)
}

func newHttpReq(
	reqTimeout contracts.RequestTimeoutDuration,
	reqHeadersTimeout contracts.RequestHeadersTimeoutDuration,
	idleConTimeout contracts.IdleConnTimeoutDuration,
) contracts.HttpReq {
	tr := &http.Transport{
		DialContext: (&net.Dialer{ //DNS resolution and connection establishment
			Timeout:   reqTimeout.GetDuration(), // DialContext.Timeout → only limits how long it takes to connect.
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   3 * time.Second,                 // TLS handshake timeout
		ResponseHeaderTimeout: reqHeadersTimeout.GetDuration(), // Wait for response headers
		ExpectContinueTimeout: 1 * time.Second,                 // Wait for 100 Continue
		IdleConnTimeout:       idleConTimeout.GetDuration(),    // Idle connection timeout
	}
	client := &http.Client{Transport: tr, Timeout: reqTimeout.GetDuration()}
	return &httpReq{client}
}
