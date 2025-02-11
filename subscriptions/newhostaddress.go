package subscriptions

import (
	"net"
	"pulsyflux/channel"
	"strconv"

	"github.com/google/uuid"
)

const (
	NewHostAddress subscId = "d0180947-2c4f-4b29-a6eb-ce3d9916580e"
)

type HostAddress struct {
	Host string
	Port int
}

func SubscribeToNewHostAddress(chnlId uuid.UUID, receive func(addr *HostAddress)) {
	channel.Subscribe(NewHostAddress.Id(), chnlId, func(addr string) {
		hostStr, portStr, _err := net.SplitHostPort(addr)
		if _err != nil {
			channel.Publish(chnlId, _err)
			return
		}
		port, convErr := strconv.Atoi(portStr)
		if convErr != nil {
			channel.Publish(chnlId, convErr)
			return
		}
		receive(&HostAddress{hostStr, port})
	})
}

func PublishNewHostAddress(chnlId uuid.UUID, address string) {
	hostStr, portStr, err := net.SplitHostPort(address)
	if err != nil {
		channel.Publish(chnlId, err)
	} else {
		port, convErr := strconv.Atoi(portStr)
		if convErr != nil {
			channel.Publish(chnlId, err)
		} else {
			hostAddr := &HostAddress{hostStr, port}
			channel.Publish(chnlId, hostAddr)
		}
	}
}

func (add *HostAddress) String() string {
	return add.Host + ":" + strconv.Itoa(add.Port)
}
