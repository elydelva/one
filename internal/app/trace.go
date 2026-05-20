package app

import (
	"context"
	"errors"
)

// TraceInput holds parameters for the ShowTrace use case.
type TraceInput struct {
	Follow bool
	Limit  int
}

// TraceEntry is a single audit log entry.
type TraceEntry struct {
	TraceID   string
	Timestamp string
	Service   string
	Action    string
	Exit      int
}

// ShowTrace reads audit log entries.
type ShowTrace struct{}

// NewShowTrace creates a ShowTrace use case.
func NewShowTrace() *ShowTrace { return &ShowTrace{} }

// Run returns trace entries.
func (uc *ShowTrace) Run(_ context.Context, _ TraceInput) ([]TraceEntry, error) {
	return nil, errors.New("not implemented")
}
