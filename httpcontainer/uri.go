package httpcontainer

import (
	"strconv"
)

type uri struct {
	protocol *string
	host     *string
	path     *string
	port     *int
}

func (u *uri) GetProtocol() *string {
	return u.protocol
}
func (u *uri) GetHost() *string {
	return u.host
}
func (u *uri) GetPath() *string {
	return u.path
}
func (u *uri) GetPort() *int {
	return u.port
}

func (u *uri) SetProtocol(protocol *string) {
	if u.protocol == nil {
		u.protocol = protocol // use the pointer passed in
		return
	}
	*(u.protocol) = *protocol
}

func (u *uri) SetHost(host *string) {
	if u.host == nil {
		u.host = host // use the pointer passed in
		return
	}
	*(u.host) = *host
}

func (u *uri) SetPort(port *int) {
	if u.port == nil {
		u.port = port // use the pointer passed in
		return
	}
	*(u.port) = *port
}

func (u *uri) SetPath(path *string) {
	if u.path == nil {
		u.path = path // use the pointer passed in
		return
	}
	*(u.path) = *path
}

func (u *uri) GetPortStr() string {
	return strconv.Itoa(*u.port)
}

func (u *uri) GetHostAddress() string {
	return *u.GetHost() + ":" + u.GetPortStr()
}

func (u *uri) String() string {
	portStr := ":"
	if *u.port > 0 {
		portStr = portStr + u.GetPortStr()
	}
	return *u.protocol + "://" + *u.host + portStr + "/" + *u.path
}
