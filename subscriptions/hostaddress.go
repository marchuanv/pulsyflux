package subscriptions

import (
	"net"
	"pulsyflux/channel"
	"strconv"
)

var HostAddressSubscription = channel.NewSubId("d0180947-2c4f-4b29-a6eb-ce3d9916580e")

type HostAddress struct {
	Host string
	Port int
}

func SubscribeToHostAddress(chnlId channel.ChnlId, receive func(addr *HostAddress)) {
	channel.Subscribe(HostAddressSubscription, chnlId, func(addr *HostAddress) {
		receive(addr)
	})
}

func PublishNewHostAddress(chnlId channel.ChnlId, address string) {
	hostStr, portStr, err := net.SplitHostPort(address)
	if err == nil {
		port, convErr := strconv.Atoi(portStr)
		if convErr == nil {
			hostAddr := &HostAddress{hostStr, port}
			channel.Publish(chnlId, hostAddr)
			return
		}
		channel.Publish(chnlId, convErr)
	}
	channel.Publish(chnlId, err)
}

func (add *HostAddress) String() string {
	return add.Host + ":" + strconv.Itoa(add.Port)
}
