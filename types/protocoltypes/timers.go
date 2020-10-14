package protocoltypes

import (
	"time"

	"github.com/nm-morais/go-babel/pkg/timer"
)

const redialTimerID = 100

type redialTimer struct {
	deadline time.Time
}

func NewRedialTimer(duration time.Duration) timer.Timer {
	return &redialTimer{
		deadline: time.Now().Add(duration),
	}
}

func (t *redialTimer) ID() timer.ID {
	return redialTimerID
}

func (t *redialTimer) Deadline() time.Time {
	return t.deadline
}

//

const flushTimerID = 101

type flushTimer struct {
	deadline time.Time
}

func NewFlushTimer(duration time.Duration) timer.Timer {
	return &flushTimer{
		deadline: time.Now().Add(duration),
	}
}

func (t *flushTimer) ID() timer.ID {
	return flushTimerID
}

func (t *flushTimer) Deadline() time.Time {
	return t.deadline
}
