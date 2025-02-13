package events

import (
	"pulsyflux/channel"

	"github.com/google/uuid"
)

var subscribeCallId = "0b7ef764-d526-4103-9440-7767db45bb52"
var convertCallId = "88e3adf3-7efc-4af5-8a12-69ff2a705fcb"

var eventChnl = channel.NewChnlId("be20fe08-7d69-4303-9dc4-96c6e285e672")

type Event[pubT any, subT any] string

type convertEvent[pubT any] struct {
	id      uuid.UUID
	pubData func() pubT
}

type publishedEvent[subT any] struct {
	id      uuid.UUID
	subData func() subT
}

func (ev Event[pubT, subT]) Subscribe(raised func(data subT)) {
	if !channel.HasChnl(eventChnl) {
		channel.OpenChnl(eventChnl)
	}
	channel.Subscribe(eventChnl, func(pubEv *publishedEvent[subT]) {
		if pubEv.id.String() == string(ev) {
			raised(pubEv.subData())
		}
	})
}

func (ev Event[pubT, subT]) Convert(conv func(data pubT) subT) {
	channel.Subscribe(eventChnl, func(conEv *convertEvent[pubT]) {
		if conEv.id.String() == string(ev) {
			channel.Publish(eventChnl, &publishedEvent[subT]{
				conEv.id,
				func() subT { return conv(conEv.pubData()) },
			})
		}
	})
}

func (ev Event[pubT, subT]) Publish(data pubT) {
	channel.Publish(eventChnl, &convertEvent[pubT]{
		uuid.MustParse(string(ev)),
		func() pubT { return data },
	})
}
