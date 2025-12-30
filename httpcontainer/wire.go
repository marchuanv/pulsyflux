//go:build wireinject
// +build wireinject

package httpcontainer

import (
	"pulsyflux/contracts"

	"time"

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
		newDefaultIdleConnTimeoutDuration,
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

func InitialiseHttpReq(idleConTimeout time.Duration) contracts.HttpReq {
	wire.Build(
		newDefaultIdleConnTimeoutDuration,
		newHttpReq,
	)
	return nil
}
