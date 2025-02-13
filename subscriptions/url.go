package subscriptions

import (
	"net/url"
	"pulsyflux/events"
)

func URLSubscription(ctor func(url *url.URL)) { events.URLEvent.Subscribe(ctor) }

func URLSubscription(ctor func(url *url.URL)) { events.URLEvent.Subscribe(ctor) }
