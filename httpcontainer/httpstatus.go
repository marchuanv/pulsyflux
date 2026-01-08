package httpcontainer

import "pulsyflux/shared"

type httpStatus int

var httpStatusText = map[httpStatus]string{
	200: "Success",
	201: "Created",
	400: "Bad Request",
	404: "Not Found",
	500: "Internal Server Error",
	504: "Gateway Timeout",
}

func (h httpStatus) Code() int {
	return int(h)
}

func (h httpStatus) String() string {
	if text, ok := httpStatusText[h]; ok {
		return text
	}
	return "Unknown Status"
}

func newHttpStatus(code int) shared.HttpStatus {
	return httpStatus(code)
}
