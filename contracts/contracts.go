package contracts

type URI interface {
	Protocol() string
	Host() string
	Port() int
	PortStr() string
	Path() string
	String() string
}

type Envelope interface {
	Content() any
}

type Connection interface {
	Open()
	Close()
	Receive(recv func(envelope Envelope))
	Send(envelope Envelope)
}

type ConnectionState interface {
	IsOpen() bool
	IsClosed() bool
}
