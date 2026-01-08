//go:build wireinject
// +build wireinject

package msgbuscontainer

import (
	"pulsyflux/contracts"

	"github.com/google/wire"
)

func InitialiseMsgBus(
	protocol uriProto,
	host uriHost,
	port uriPort,
	path uriPath,
) contracts.MsgBus {
	wire.Build(
		wire.Struct(new(uri), "*"),
		wire.Bind(new(contracts.URI), new(*uri)),
		newChannel,
		newMsgBus,
	)
	return nil
}
