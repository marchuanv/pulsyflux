package httpcontainer

import (
	"net/http"
	"pulsyflux/contracts"
	"pulsyflux/util"

	"github.com/google/uuid"
)

type httpReq struct {
	client *http.Client
}

func (r *httpReq) Send(addr contracts.URI, msgId uuid.UUID, content string) string {
	_content := util.ReaderFromString(content)
	resp, err := r.client.Post(addr.String(), "application/json", _content)
	if err != nil {
		panic(err)
	}
	return util.StringFromReader(resp.Body)
}

func newHttpReq(idleConTimeout contracts.IdleConnTimeoutDuration) contracts.HttpReq {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    idleConTimeout.GetDuration(),
		DisableCompression: true,
		DisableKeepAlives:  true,
	}
	client := &http.Client{Transport: tr}
	return &httpReq{client}
}
