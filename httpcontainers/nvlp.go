package httpcontainers

import (
	"net/url"
)

type envelope struct {
	url *url.URL
	msg func() any
}

func (nvlp *envelope) GetUrl() *url.URL {
	return nvlp.url
}

func (nvlp *envelope) SetUrl(url *url.URL) {
	nvlp.url = url
}

func (nvlp *envelope) SetMsg(msg any) {
	nvlp.msg = func() any {
		return msg
	}
}

func (nvlp *envelope) GetMsg() any {
	return nvlp.msg()
}
