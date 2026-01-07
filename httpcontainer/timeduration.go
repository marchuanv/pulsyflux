package httpcontainer

import (
	"pulsyflux/contracts"
	"time"
)

type timeDuration struct {
	duration time.Duration
}

func (d *timeDuration) GetDuration() time.Duration {
	return d.duration
}

func newDefaultWriteTimeoutDuration() contracts.WriteTimeDuration { //limits how long the server can write the response.
	return &timeDuration{
		duration: 5 * time.Second,
	}
}

func newDefaultReadTimeoutDuration() contracts.ReadTimeDuration { //limits how long the server waits for the request body to arrive.
	return &timeDuration{
		duration: 5 * time.Second,
	}
}

func newDefaultClientIdleConnTimeoutDuration() contracts.IdleConnTimeoutDuration { //limits how long idle connections stay open.
	return &timeDuration{
		duration: 25 * time.Second,
	}
}

func newDefaultServerIdleConnTimeoutDuration() contracts.IdleConnTimeoutDuration { //limits how long idle connections stay open.
	return &timeDuration{
		duration: 25 * time.Second,
	}
}

func newDefaultClientRequestTimeoutDuration() contracts.RequestTimeoutDuration {
	return &timeDuration{
		duration: 25 * time.Second,
	}
}

func newDefaultServerResponseTimeoutDuration() contracts.ResponseTimeoutDuration {
	return &timeDuration{
		duration: 25 * time.Second,
	}
}

func newDefaultClientRequestHeadersTimeoutDuration() contracts.RequestHeadersTimeoutDuration {
	return &timeDuration{
		duration: 0, //disable
	}
}
