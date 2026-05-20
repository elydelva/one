package ports

import "time"

// Clock provides the current time. Injected to make time deterministic in tests.
type Clock interface {
	Now() time.Time
}
