package httpcontainers

import (
	"strconv"
)

type uri struct {
	protocol string
	host     string
	path     string
	port     int
}

func (u *uri) GetScheme() string {
	return u.protocol
}
func (u *uri) GetHost() string {
	return u.host
}
func (u *uri) GetPath() string {
	return u.path
}
func (u *uri) GetPort() int {
	return u.port
}

func (u *uri) SetScheme(scheme string) {
	u.protocol = scheme
}
func (u *uri) SetHost(host string) {
	u.host = host
}
func (u *uri) SetPath(path string) {
	u.path = path
}
func (u *uri) SetPort(port int) {
	u.port = port
}

func (u *uri) GetPortStr() string {
	return strconv.Itoa(u.port)
}

func (u *uri) String() string {
	portStr := ":"
	if u.port > 0 {
		portStr = portStr + u.GetPortStr()
	}
	return u.protocol + ":" + u.host + portStr + "/" + u.path
}
