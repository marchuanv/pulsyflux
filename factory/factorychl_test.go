package factory

import (
	"net/url"
	"pulsyflux/contracts"
	"testing"
	"time"
)

func TestFactoryChl(test *testing.T) {
	var nvlp *contracts.Envelope
	urlFact := RegisterURLFactory()
	urlFact.get(func(url *url.URL) {
		nvlpFact := RegisterEnvlpFactory()
		nvlpFact.get(func(obj contracts.Envelope) {
			nvlp = &obj
		}, Arg{"url", url}, Arg{"msg", "Hello World"})
	}, Arg{"urlStr", "www.google.com"})
	time.Sleep(1000 * time.Millisecond)
	if nvlp == nil {
		test.Log("expected an envelope object")
		test.Fail()
	}
}
