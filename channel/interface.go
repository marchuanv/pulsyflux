package channel

const (
	ChannelReadReady  ChannelEvent = "c8227ac9-dfe7-4ed3-bdbc-ddcd685b506e"
	ChannelWriteReady ChannelEvent = "017f54b4-4279-4a9c-8248-be380b58c9a6"
	ChannelClosed     ChannelEvent = "acad55dd-8b47-436b-b5b1-3ad5995704f9"
	ChannelError      ChannelEvent = "40cd6186-6d9f-4fce-b062-20dd952f03ac"
	ChannelRead       ChannelEvent = "d57d9ca5-4c75-47fe-b597-11f71852e08b"
	ChannelReadOnly   ChannelEvent = "a2a96f54-0248-4eff-8911-582a9983e080"
)

type ChannelEvent string
type Channel[T any] interface {
	Send(msg Msg[T])
	OnSent()
	Receive() Msg[T]
	OnReceived()
	Close()
	OnClosed()
	SetReadOnly()
}

var glbChnlEventLookup = make(map[ChannelEvent]Msg[ChannelEvent])

func ensureGlbChnlEventLookup() {
	if len(glbChnlEventLookup) == 0 {
		glbChnlEventLookup[ChannelReadReady] = newChnlMsg(ChannelReadReady)
		glbChnlEventLookup[ChannelWriteReady] = newChnlMsg(ChannelWriteReady)
		glbChnlEventLookup[ChannelClosed] = newChnlMsg(ChannelClosed)
		glbChnlEventLookup[ChannelError] = newChnlMsg(ChannelError)
		glbChnlEventLookup[ChannelRead] = newChnlMsg(ChannelRead)
		glbChnlEventLookup[ChannelReadOnly] = newChnlMsg(ChannelReadOnly)
	}
}
