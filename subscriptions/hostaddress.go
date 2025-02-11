package subscriptions

import (
	"net"
	"pulsyflux/channel"
	"strconv"
)

type HostAddress struct {
	Host string
	Port int
}

func SubscribeToHostAddress(chnlId channel.ChnlId, receive func(addr *HostAddress)) {
	channel.Subscribe(chnlId, func(addr *HostAddress) {
		receive(addr)
	})
}

func PublishHostAddress(chnlId channel.ChnlId, address string) {
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
