package connect

import (
	"fmt"
)

type HttpSchema int
type HttpMethod int

const (
	HTTPSchema HttpSchema = iota
	HTTPSSchema
	HttpGET HttpMethod = iota
	HttpPOST
	NO_RESPONSE
)

func (sch HttpSchema) String() string {
	switch sch {
	case HTTPSchema:
		return "http"
	case HTTPSSchema:
		return "https"
	default:
		return fmt.Sprintf("%d", int(sch))
	}
}

func (method HttpMethod) String() string {
	switch method {
	case HttpGET:
		return "GET"
	case HttpPOST:
		return "POST"
	default:
		return fmt.Sprintf("%d", int(method))
	}
}
