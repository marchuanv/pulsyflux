package channel

type chnlErr struct {
	msg string
}

func newChnlError(msg string) error {
	return &chnlErr{msg}
}

func (err *chnlErr) Error() string {
	return err.msg
}

func isChnlError(err any) (converted error, canConvert bool) {
	defer (func() {
		if recover() != nil {
			canConvert = false
		}
	})()
	converted = err.(*chnlErr)
	canConvert = true
	return converted, canConvert
}

func isError[T error](err any) (converted error, canConvert bool) {
	defer (func() {
		if recover() != nil {
			canConvert = false
		}
	})()
	converted = err.(T)
	canConvert = true
	return converted, canConvert
}
