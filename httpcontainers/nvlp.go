package httpcontainers

import (
	"net/url"
)

type envelope struct {
	url *url.URL
	msg *string
}

func (nvlp *envelope) GetUrl() *url.URL {
	return nvlp.url
}

func (nvlp *envelope) SetUrl(url *url.URL) {
	if nvlp.url == nil {
		nvlp.url = url // use the pointer passed in
		return
	}
	*(nvlp.url) = *url
}

func (nvlp *envelope) SetMsg(msg *string) {
	if nvlp.msg == nil {
		nvlp.msg = msg // use the pointer passed in
		return
	}
	*(nvlp.msg) = *msg
}

func (nvlp *envelope) GetMsg() *string {
	return nvlp.msg
}
