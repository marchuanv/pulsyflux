package httpcontainer

import (
	"pulsyflux/shared"
	"time"
)

type timeDuration struct {
	duration time.Duration
}

func (d *timeDuration) GetDuration() time.Duration {
	return d.duration
}

func newDefaultWriteTimeoutDuration() shared.WriteTimeDuration { //limits how long the server can write the response.
	return &timeDuration{
		duration: 5 * time.Second,
	}
}

func newDefaultReadTimeoutDuration() shared.ReadTimeDuration { //limits how long the server waits for the request body to arrive.
	return &timeDuration{
		duration: 5 * time.Second,
	}
}

func newDefaultClientIdleConnTimeoutDuration() shared.IdleConnTimeoutDuration { //limits how long idle connections stay open.
	return &timeDuration{
		duration: 25 * time.Second,
	}
}

func newDefaultServerIdleConnTimeoutDuration() shared.IdleConnTimeoutDuration { //limits how long idle connections stay open.
	return &timeDuration{
		duration: 25 * time.Second,
	}
}

func newDefaultClientRequestTimeoutDuration() shared.RequestTimeoutDuration {
	return &timeDuration{
		duration: 25 * time.Second,
	}
}

func newDefaultServerResponseTimeoutDuration() shared.ResponseTimeoutDuration {
	return &timeDuration{
		duration: 25 * time.Second,
	}
}

func newDefaultClientRequestHeadersTimeoutDuration() shared.RequestHeadersTimeoutDuration {
	return &timeDuration{
		duration: 0, //disable
	}
}
