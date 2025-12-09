package httpcontainer

import (
	"time"
)

type timeDuration struct {
	duration time.Duration
}

func (d *timeDuration) SetDuration(duration time.Duration) {
	d.duration = duration
}

func (d *timeDuration) GetDuration() time.Duration {
	return d.duration
}
