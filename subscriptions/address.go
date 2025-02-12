package subscriptions

import (
	"net"
	"pulsyflux/channel"
	"strconv"
)

type Address struct {
	Host string
	Port int
}

func SubscribeToAddress(chnlId channel.ChnlId, receive func(addr *Address)) {
	channel.Subscribe(chnlId, func(addr *Address) {
		receive(addr)
	})
}

func PublishAddress(chnlId channel.ChnlId, address string) {
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

func (add *Address) String() string {
	return add.Host + ":" + strconv.Itoa(add.Port)
}
