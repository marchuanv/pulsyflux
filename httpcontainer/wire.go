//go:build wireinject
// +build wireinject

package httpcontainer

import (
	"pulsyflux/shared"

	"github.com/google/uuid"
	"github.com/google/wire"
)

func InitialiseHttpServer(
	protocol uriProto,
	host uriHost,
	port uriPort,
	path uriPath,
) shared.HttpServer {
	wire.Build(
		wire.Struct(new(uri), "*"),
		wire.Bind(new(shared.URI), new(*uri)),
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

func InitialiseHttpResHandler(msgUd uuid.UUID, successStatusCode int) shared.HttpResHandler {
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

func InitialiseHttpReq2() shared.HttpReq {
	wire.Build(
		newDefaultClientRequestTimeoutDuration,
		newDefaultClientRequestHeadersTimeoutDuration,
		newDefaultClientIdleConnTimeoutDuration,
		newHttpReq,
	)
	return nil
}
