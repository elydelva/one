package fake

import (
	"sync"
	"time"

	"elydelva/one/internal/ports"
)

// Clock is a controllable clock for testing.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

// NewClock creates a fake clock set to the given time.
// If no time is given, defaults to 2024-01-01 00:00:00 UTC.
func NewClock(t ...time.Time) *Clock {
	if len(t) > 0 {
		return &Clock{now: t[0]}
	}
	return &Clock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by d.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// Set moves the clock to the given time.
func (c *Clock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

var _ ports.Clock = (*Clock)(nil)
