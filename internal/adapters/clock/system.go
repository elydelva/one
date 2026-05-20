package clock

import (
	"time"

	"elydelva/one/internal/ports"
)

// SystemClock returns the real current time.
type SystemClock struct{}

// NewSystemClock creates a clock backed by the system time.
func NewSystemClock() *SystemClock { return &SystemClock{} }

func (c *SystemClock) Now() time.Time { return time.Now() }

var _ ports.Clock = (*SystemClock)(nil)
