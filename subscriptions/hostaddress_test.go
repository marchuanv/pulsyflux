package subscriptions

import (
	"pulsyflux/channel"
	"testing"
	"time"
)

func TestHostAddress(test *testing.T) {
	chnlId := channel.NewChnlId("58ff24d9-1de4-49ba-bd9d-7cc400e1bbdf")
	channel.OpenChnl(chnlId)
	subRslvd := false
	SubscribeToErrors(chnlId, func(err error) {
		test.Log(err)
		test.Fail()
	})
	SubscribeToHostAddress(chnlId, func(addr *HostAddress) {
		subRslvd = true
	})
	PublishHostAddress(chnlId, "localhost:3000")
	time.Sleep(1000 * time.Millisecond)
	if !subRslvd {
		test.Fail()
	}
}
