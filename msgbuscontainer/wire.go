//go:build wireinject
// +build wireinject

package msgbuscontainer

import (
	"pulsyflux/httpcontainer"
	"pulsyflux/shared"

	"github.com/google/uuid"
	"github.com/google/wire"
)

func InitialiseMsgBus(
	protocol shared.URIProtocol,
	host shared.URIHost,
	port shared.URIPort,
	path shared.URIPath,
	channelId uuid.UUID,
	successStatusCode int,
) shared.MsgBus {
	wire.Build(
		shared.NewUri,
		newChannel,
		httpcontainer.InitialiseHttpResHandler,
		httpcontainer.InitialiseHttpReq,
		newMsgBus,
	)
	return nil
}
