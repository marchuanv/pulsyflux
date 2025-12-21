package httpcontainer

import (
	"pulsyflux/contracts"
	"strconv"
)

type protocol string
type host string
type path string
type port int

type uri struct {
	protocol protocol
	host     host
	path     path
	port     port
}

func (u *uri) GetPortStr() string {
	return strconv.Itoa(int(u.port))
}

func (u *uri) GetHostAddress() string {
	return string(u.host) + ":" + u.GetPortStr()
}

func (u *uri) String() string {
	portStr := ":"
	if u.port > 0 {
		portStr = portStr + u.GetPortStr()
	}
	return string(u.protocol) + "://" + string(u.host) + portStr + "/" + string(u.path)
}

func newUri(
	protocol protocol,
	host host,
	port port,
	path path,
) contracts.URI {
	return &uri{
		protocol: protocol,
		host:     host,
		port:     port,
		path:     path,
	}
}
