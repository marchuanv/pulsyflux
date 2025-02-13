package events

import (
	"net/url"
)

type URI interface {
	Host() string
	Port() string
	Path() string
	String() string
}

const (
	URLEvent Event[URI, *url.URL] = "725fce3e-4215-4a8b-b7cf-2af95c9be727"
)
