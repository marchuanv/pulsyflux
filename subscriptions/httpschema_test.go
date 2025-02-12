package subscriptions

import (
	"pulsyflux/channel"
	"testing"
	"time"
)

func TestHttpSchema(test *testing.T) {
	chnlId := channel.NewChnlId("4fefa6ae-c026-4df3-980c-9d76e7121d83")
	channel.OpenChnl(chnlId)
	subRslvd := false
	SubscribeToErrors(chnlId, func(err error) {
		test.Log(err)
		test.Fail()
	})
	SubscribeToHttpSchema(chnlId, func(httpSchema string) {
		subRslvd = true
	})
	PublishHttpSchema(chnlId, HTTP)
	time.Sleep(1000 * time.Millisecond)
	if !subRslvd {
		test.Fail()
	}
}
