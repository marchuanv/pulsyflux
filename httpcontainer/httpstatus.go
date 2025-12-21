package httpcontainer

import "pulsyflux/contracts"

type httpStatus int

var httpStatusText = map[httpStatus]string{
	200: "Success",
	201: "Created",
	500: "Internal Server Error",
}

func (h httpStatus) Code() int {
	return int(h)
}

func (h httpStatus) String() string {
	return httpStatusText[h]
}

func newHttpStatus(code int) contracts.HttpStatus {
	return httpStatus(code)
}
