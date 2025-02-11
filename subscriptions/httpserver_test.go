package subscriptions

import (
	"net"
	"net/http"
	"pulsyflux/channel"
	"testing"
	"time"
)

func TestHttpServer(test *testing.T) {
	chnlId := channel.NewChnlId("483911ea-d9d9-436c-ac3f-1d78be62723a")
	channel.OpenChnl(chnlId)
	subRslvd := false
	SubscribeToErrors(chnlId, func(err error) {
		test.Log(err)
		test.Fail()
	})
	SubscribeToHttpServer(chnlId, func(listener net.Listener, server *http.Server, addr *HostAddress) {
		subRslvd = true
	})
	PublishHttpServer(chnlId)
	PublishHttpListener(chnlId)
	PublishHostAddress(chnlId, "localhost:3000")
	time.Sleep(2000 * time.Millisecond)
	if !subRslvd {
		test.Fail()
	}
}
