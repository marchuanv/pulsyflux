package httpcontainer

import (
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/sliceext"
	"time"

	"github.com/google/uuid"
)

const (
	httpRequestHandlerId          contracts.TypeId[httpReqHandler]                                          = "02c88068-3a66-4856-b2bf-e2dce244761b"
	httpRequestHandlerResponsesId contracts.TypeId[sliceext.List[contracts.TypeId[contracts.HttpResponse]]] = "172c2cd8-4869-43a7-aa1a-af341e0b439f"

	HttpServerId               contracts.TypeId[httpServer]   = "fccaca3e-54a4-400d-a9d3-b80a2161c20f"
	HttpServerAddressId        contracts.TypeId[uri]          = "467be880-a887-42dd-b6f8-49591a47c5da"
	httpServerReadTimeoutId    contracts.TypeId[timeDuration] = "49e0575b-a6bc-45d1-aba0-4cf57c48fc06"
	httpServerWriteTimeoutId   contracts.TypeId[timeDuration] = "1958180b-bd2d-4691-8a3f-50a35b5dbaf5"
	httpServerMaxHeaderBytesId contracts.TypeId[int]          = "a22a6899-499f-4779-b66b-75952f4d766a"
)

func uriConfig(uriTypeId contracts.TypeId[uri], protocol *string, host *string, port *int, path *string) {
	protocolTypeId := contracts.TypeId[string](uuid.NewString())
	hostTypeId := contracts.TypeId[string](uuid.NewString())
	pathTypeId := contracts.TypeId[string](uuid.NewString())
	portTypeId := contracts.TypeId[int](uuid.NewString())
	containers.RegisterType(uriTypeId)
	containers.RegisterTypeDependency(uriTypeId, protocolTypeId, "protocol", protocol)
	containers.RegisterTypeDependency(uriTypeId, hostTypeId, "host", host)
	containers.RegisterTypeDependency(uriTypeId, pathTypeId, "path", path)
	containers.RegisterTypeDependency(uriTypeId, portTypeId, "port", port)
}

func init() {

	protocol := "http"

	uriConfig(HttpServerAddressId, &protocol, nil, nil, nil)

	containers.RegisterType(HttpServerId)
	containers.RegisterTypeDependency(HttpServerId, HttpServerAddressId, "address", nil)
	timeDuration := &timeDuration{duration: 10 * time.Second}
	containers.RegisterTypeDependency(HttpServerId, httpServerReadTimeoutId, "readTimeout", timeDuration)
	containers.RegisterTypeDependency(HttpServerId, httpServerWriteTimeoutId, "writeTimeout", timeDuration)
	maxHeaderBytes := 16 * 1024 //16KB (16 * 1024 bytes)
	containers.RegisterTypeDependency(HttpServerId, httpServerMaxHeaderBytesId, "maxHeaderBytes", &maxHeaderBytes)

	containers.RegisterType(httpRequestHandlerId)
	httpResponseIds := sliceext.NewList[contracts.TypeId[contracts.HttpResponse]]()
	containers.RegisterTypeDependency(httpRequestHandlerId, httpRequestHandlerResponsesId, "httpResponseIds", httpResponseIds)

	containers.RegisterTypeDependency(HttpServerId, httpRequestHandlerId, "handler", nil)
}

func HttpResponseConfig(httpResTypeId contracts.TypeId[contracts.HttpResponse], msgId contracts.MsgId[contracts.Msg], successStatusCode int, successStatusMsg string) {

	responseTypeId := contracts.TypeId[httpResHandler](httpResTypeId)
	httpResponseIds := containers.Get[*sliceext.List[contracts.TypeId[contracts.HttpResponse]]](httpRequestHandlerResponsesId)
	httpResponseIds.Add(httpResTypeId)

	msgTypeMsgTypeId := contracts.TypeId[contracts.MsgId[contracts.Msg]](uuid.NewString())
	succStCoTypeId := contracts.TypeId[int](uuid.NewString())
	succStMsgTypeId := contracts.TypeId[string](uuid.NewString())

	incMsgTypId := contracts.TypeId[chan contracts.Msg](uuid.NewString())
	outMsgTypId := contracts.TypeId[chan contracts.Msg](uuid.NewString())

	containers.RegisterType(responseTypeId)
	containers.RegisterTypeDependency(responseTypeId, msgTypeMsgTypeId, "msgId", &msgId)
	containers.RegisterTypeDependency(responseTypeId, succStCoTypeId, "successStatusCode", &successStatusCode)
	containers.RegisterTypeDependency(responseTypeId, succStMsgTypeId, "successStatusMsg", &successStatusMsg)

	intCh := make(chan contracts.Msg)
	outCh := make(chan contracts.Msg)

	containers.RegisterTypeDependency(responseTypeId, incMsgTypId, "incMsg", &intCh)
	containers.RegisterTypeDependency(responseTypeId, outMsgTypId, "outMsg", &outCh)
}
