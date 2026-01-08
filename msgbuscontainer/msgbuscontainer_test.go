package msgbuscontainer

import (
	"pulsyflux/shared"
	"testing"

	"github.com/google/uuid"
)

func TestMsgBus(test *testing.T) {
	protocol := "http"
	host := "localhost"
	port := 3000
	path := "/"
	InitialiseMsgBus(
		shared.URIProtocol(protocol),
		shared.URIHost(host),
		shared.URIPort(port),
		shared.URIPath(path),
		uuid.New(),
		200,
	)
}
