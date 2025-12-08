package factory

import (
	"testing"
)

func TestFactoryChl(test *testing.T) {
	urlFact := RegisterURLFactory()
	url := urlFact.get(Arg{"urlStr", "www.google.com"})
	nvlpFact := RegisterEnvlpFactory()
	nvlp := nvlpFact.get(Arg{"url", url}, Arg{"msg", "Hello World"})
	if nvlp == nil {
		test.Log("expected an envelope object")
		test.Fail()
	}
}
