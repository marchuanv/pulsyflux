//go:build wireinject
// +build wireinject

package httpcontainer

import (
	"pulsyflux/shared"

	"github.com/google/uuid"
	"github.com/google/wire"
)

func InitialiseHttpServer(
	protocol shared.URIProtocol,
	host shared.URIHost,
	port shared.URIPort,
	path shared.URIPath,
) shared.HttpServer {
	wire.Build(
		shared.NewUri,
		newDefaultReadTimeoutDuration,
		newDefaultWriteTimeoutDuration,
		newDefaultServerIdleConnTimeoutDuration,
		newDefaultServerResponseTimeoutDuration,
		newDefaultHttpMaxHeaderBytes,
		newHttpReqHandler,
		newHttpServer,
	)
	return nil
}

func InitialiseHttpResHandler(msgId uuid.UUID, successStatusCode int) shared.HttpResHandler {
	wire.Build(
		newHttpMsgId,
		newHttpStatus,
		newHttpResHandler,
	)
	return nil
}

func InitialiseHttpReq() shared.HttpReq {
	wire.Build(
		newDefaultClientRequestTimeoutDuration,
		newDefaultClientRequestHeadersTimeoutDuration,
		newDefaultClientIdleConnTimeoutDuration,
		newHttpReq,
	)
	return nil
}
