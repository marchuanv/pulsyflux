package subscriptions

import (
	"pulsyflux/channel"
	"testing"
	"time"
)

func TestHttpRequestMethod(test *testing.T) {
	chnlId := channel.NewChnlId("3260d0aa-65d1-43e5-8169-9e1bf9f08678")
	channel.OpenChnl(chnlId)
	subRslvd := false
	SubscribeToErrors(chnlId, func(err error) {
		test.Log(err)
		test.Fail()
	})
	SubscribeToHttpRequestMethod(chnlId, func(httpMethod string) {
		subRslvd = true
	})
	PublishHttpRequestMethod(chnlId, HttpGET)
	time.Sleep(1000 * time.Millisecond)
	if !subRslvd {
		test.Fail()
	}
}
