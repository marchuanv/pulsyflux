package httpcontainer

import (
	"pulsyflux/contracts"
	"strconv"
)

type uriProto string
type uriHost string
type uriPath string
type uriPort int

type uri struct {
	protocol uriProto
	host     uriHost
	path     uriPath
	port     uriPort
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
	protocol uriProto,
	host uriHost,
	port uriPort,
	path uriPath,
) contracts.URI {
	return &uri{
		protocol: protocol,
		host:     host,
		port:     port,
		path:     path,
	}
}
