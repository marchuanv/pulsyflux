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

func newTimeDuration(duration time.Duration) contracts.TimeDuration {
	return &timeDuration{
		duration: duration,
	}
}

func newDefaultWriteTimeoutDuration() contracts.WriteTimeDuration {
	return &timeDuration{
		duration: 10 * time.Second,
	}
}

func newDefaultReadTimeoutDuration() contracts.ReadTimeDuration {
	return &timeDuration{
		duration: 10 * time.Second,
	}
}
