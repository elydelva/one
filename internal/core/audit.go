package core

import "time"

// AuditKind enumerates audit event categories.
type AuditKind string

const (
	AuditLogin   AuditKind = "LOGIN"
	AuditLogout  AuditKind = "LOGOUT"
	AuditRefresh AuditKind = "REFRESH"
	AuditExec    AuditKind = "EXEC"
	AuditScope   AuditKind = "SCOPE_CHANGE"
	AuditRotate  AuditKind = "ROTATE"
)

// AuditOutcome represents the result of an audited operation.
type AuditOutcome string

const (
	OutcomeOK     AuditOutcome = "ok"
	OutcomeError  AuditOutcome = "error"
	OutcomeDenied AuditOutcome = "denied"
)

// HTTPCall records an outbound HTTP request made during action execution.
// Bodies are intentionally excluded; secrets MUST never appear here.
type HTTPCall struct {
	Method     string `json:"method"`
	URL        string `json:"url"`
	Status     int    `json:"status"`
	DurationMS int64  `json:"duration_ms"`
}

// AuditEvent is a single audit log entry. Secrets MUST never appear in this struct.
type AuditEvent struct {
	TraceID   string       `json:"trace_id,omitempty"`
	At        time.Time    `json:"at"`
	Kind      AuditKind    `json:"kind"`
	Service   string       `json:"service,omitempty"`
	Account   string       `json:"account,omitempty"`
	Action    string       `json:"action,omitempty"`
	Outcome   AuditOutcome `json:"outcome"`
	Err       string       `json:"err,omitempty"`
	HTTPCalls []HTTPCall   `json:"http_calls,omitempty"`
}

// AuditFilter narrows audit log reads.
type AuditFilter struct {
	Since   time.Time
	Service string
	Kind    AuditKind
	TraceID string
	Limit   int
}
