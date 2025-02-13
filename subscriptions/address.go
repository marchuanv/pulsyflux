package subscriptions

import (
	"net"
	"pulsyflux/channel"
	"strconv"
)

func PublishAddress(chnlId channel.ChnlId, addr string) {
	hostStr, portStr, err := net.SplitHostPort(addr)
	if err == nil {
		port, convErr := strconv.Atoi(portStr)
		if convErr == nil {
			channel.Publish(chnlId, &address{hostStr, port})
		} else {
			channel.Publish(chnlId, convErr)
		}
	}
	channel.Publish(chnlId, err)
}

func (addr Address) Host() string {
	return addr.Host + ":" + strconv.Itoa(add.Port)
}

func (addr Address) Port() string {
	return addr.Host + ":" + strconv.Itoa(add.Port)
}

func (addr Address) String() string {
	return addr.Host + ":" + strconv.Itoa(add.Port)
}
