package subscriptions

import (
	"net"
	"pulsyflux/channel"
	"testing"
	"time"
)

func TestHttpListener(test *testing.T) {
	chnlId := channel.NewChnlId("18d7e9de-aa1c-430f-ac2e-3216003cbe8e")
	channel.OpenChnl(chnlId)
	subRslvd := false
	SubscribeToErrors(chnlId, func(err error) {
		test.Log(err)
		test.Fail()
	})
	SubscribeToErrors(chnlId, func(err error) {
		test.Log(err)
		test.Fail()
	})
	SubscribeToHttpListener(chnlId, func(listener net.Listener, addr *HostAddress) {
		subRslvd = true
	})
	PublishHttpListener(chnlId)
	PublishHostAddress(chnlId)
	time.Sleep(1000 * time.Millisecond)
	if !subRslvd {
		test.Fail()
	}
}
