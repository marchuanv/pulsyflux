package httpcontainer

import (
	"context"
	"errors"
	"net"
	"net/http"
	"pulsyflux/shared"
	"pulsyflux/util"
	"time"

	"github.com/google/uuid"
)

type httpReq struct {
	client *http.Client
}

func (r *httpReq) Send(
	addr shared.URI,
	msgId uuid.UUID,
	msgIn shared.Msg,
) (shared.HttpStatus, shared.Msg, error) {

	_content := msgId.String() + string(msgIn)

	// Create POST request with request body
	// util.ReaderFromString returns a plain *bytes.Reader (does not close itself)
	req, err := http.NewRequest(http.MethodPost, addr.String(), util.ReaderFromString(_content))
	if err != nil {
		// Request could not be constructed (e.g., invalid URL)
		return newHttpStatus(http.StatusBadRequest), "", err
	}

	// Execute the request
	resp, err := r.client.Do(req)
	if err != nil {
		// Timeout due to context deadline
		if errors.Is(err, context.DeadlineExceeded) {
			return newHttpStatus(http.StatusGatewayTimeout), "", err
		}

		// Network timeout (TCP, TLS handshake, etc.)
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return newHttpStatus(http.StatusGatewayTimeout), "", err
		}

		// Other transport or I/O errors
		return newHttpStatus(http.StatusInternalServerError), "", err
	}

	// Consume and close the response body using utility
	// util.StringFromReader(resp.Body) is responsible for closing resp.Body
	body, err := util.StringFromReader(resp.Body)
	if err != nil {
		return newHttpStatus(http.StatusInternalServerError), "", err
	}

	// Return server status code and body
	return newHttpStatus(resp.StatusCode), shared.Msg(body), nil
}

func newHttpReq(
	reqTimeout shared.RequestTimeoutDuration,
	reqHeadersTimeout shared.RequestHeadersTimeoutDuration,
	idleConTimeout shared.IdleConnTimeoutDuration,
) shared.HttpReq {

	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   reqTimeout.GetDuration(),
			KeepAlive: 30 * time.Second,
		}).DialContext,

		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: reqHeadersTimeout.GetDuration(),
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       idleConTimeout.GetDuration(),

		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   reqTimeout.GetDuration(),
	}

	return &httpReq{client: client}
}
