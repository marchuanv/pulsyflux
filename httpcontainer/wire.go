//go:build wireinject
// +build wireinject

package httpcontainer

import (
	"pulsyflux/contracts"

	"github.com/google/uuid"
	"github.com/google/wire"
)

func InitialiseHttpServer(
	protocol protocol,
	host host,
	port port,
	path path,
) contracts.HttpServer {
	wire.Build(
		wire.Struct(new(uri), "*"),
		wire.Bind(new(contracts.URI), new(*uri)),
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

func InitialiseHttpResHandler(msgUd uuid.UUID, successStatusCode int) contracts.HttpResHandler {
	wire.Build(
		newHttpMsgId,
		newHttpStatus,
		newHttpResHandler,
	)
	return nil
}

func InitialiseHttpReq() contracts.HttpReq {
	wire.Build(
		newDefaultClientRequestTimeoutDuration,
		newDefaultClientRequestHeadersTimeoutDuration,
		newDefaultClientIdleConnTimeoutDuration,
		newHttpReq,
	)
	return nil
}
