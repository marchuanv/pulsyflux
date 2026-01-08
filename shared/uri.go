package shared

import (
	"strconv"
)

type URIProtocol string
type URIHost string
type URIPath string
type URIPort int

type uri struct {
	protocol URIProtocol
	host     URIHost
	path     URIPath
	port     URIPort
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

func NewUri(
	protocol URIProtocol,
	host URIHost,
	port URIPort,
	path URIPath,
) URI {
	return &uri{protocol, host, path, port}
}
