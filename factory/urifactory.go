package factory

import (
	"net/url"
	"pulsyflux/contracts"
	"strconv"
)

const (
	uriFactory factory[contracts.URI] = "99047248-a20c-471d-bb10-ec1b8e528d05"
)

type uri struct {
	protocol string
	host     string
	path     string
	port     int
}

func RegisterURIFactory() factory[contracts.URI] {
	uriFactory.register(func(args ...Arg) contracts.URI {
		isUrl, url := argValue[*url.URL](&args[0])
		if isUrl {
			conv := uri{
				url.Scheme,
				url.Host,
				url.Path,
				0,
			}
			_port, err := strconv.Atoi(url.Port())
			if err != nil {
				panic(err)
			}
			conv.port = _port
			return contracts.URI(conv)
		}
		return nil
	})
	return uriFactory
}

func (u uri) Protocol() string {
	return u.protocol
}
func (u uri) Host() string {
	return u.host
}
func (u uri) Port() int {
	return u.port
}
func (u uri) PortStr() string {
	return strconv.Itoa(u.port)
}
func (u uri) Path() string {
	return u.path
}
func (u uri) String() string {
	portStr := ":"
	if u.port > 0 {
		portStr = portStr + u.PortStr()
	}
	return u.protocol + ":" + u.host + portStr + "/" + u.path
}
