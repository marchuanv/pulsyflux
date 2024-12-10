package channel

type chnlmsg[T any] func() T

type Msg[T any] interface {
	Data() T
}

func newChnlMsg[T any](data T) Msg[T] {
	var fun chnlmsg[T]
	fun = func() T {
		return data
	}
	return fun
}

func (chMsg chnlmsg[T]) Data() T {
	return chMsg()
}
