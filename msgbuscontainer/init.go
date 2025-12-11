package msgbuscontainers

import (
	"log"
	"net/http"
	"pulsyflux/containers"
	"pulsyflux/contracts"
	"pulsyflux/httpcontainer"

	"github.com/google/uuid"
)

const (
	HttpMsgBusId    contracts.TypeId[msgBus[contracts.Msg]]              = "56059ff5-1ec7-45f9-83ca-e2ba53821f85"
	httpMsgBusChlId contracts.TypeId[contracts.ChannelId[contracts.Msg]] = "cd21fb74-b8b8-450b-a173-6ad3f0ea236d"
)

func init() {
	containers.RegisterType(HttpMsgBusId)
	msgBusGlobalChl := contracts.ChannelId[contracts.Msg](uuid.NewString())
	containers.RegisterTypeDependency(
		HttpMsgBusId,
		httpMsgBusChlId,
		"chl",
		&msgBusGlobalChl,
	)
	msgId := contracts.MsgId[contracts.Msg](msgBusGlobalChl)
	httpResTypeId := contracts.TypeId[contracts.HttpResponse](msgBusGlobalChl)
	httpcontainer.HttpResponseConfig(httpResTypeId, msgId, http.StatusCreated, "subscribed to channel")
	log.Printf("global msgbus channnel(%s)", msgBusGlobalChl)
}
