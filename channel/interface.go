package channel

const (
	ChannelReadReady  ChannelEvent = "c8227ac9-dfe7-4ed3-bdbc-ddcd685b506e"
	ChannelWriteReady ChannelEvent = "017f54b4-4279-4a9c-8248-be380b58c9a6"
	ChannelClosed     ChannelEvent = "acad55dd-8b47-436b-b5b1-3ad5995704f9"
	ChannelError      ChannelEvent = "40cd6186-6d9f-4fce-b062-20dd952f03ac"
	ChannelRead       ChannelEvent = "d57d9ca5-4c75-47fe-b597-11f71852e08b"
)

type ChannelEvent string

type Channel interface {
	Read(receiveData func(data any))
	Write(data any)
	Close()
	RaiseEvent(event ChannelEvent)
	WaitForEvent(event ChannelEvent)
	HasEvent(event ChannelEvent) bool
}
