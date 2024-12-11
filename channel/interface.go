package channel

type Channel interface {
	Send(chnlMsg ChannelMsg)
	Message() ChannelMsg
	IsClosed() bool
	Close()
}
