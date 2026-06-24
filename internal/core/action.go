package core

import "encoding/json"

// ActionID identifies an action within a service (e.g. "issues.create").
type ActionID string

// HandlerRef points to a WASM handler file in the catalog and carries the
// per-action capability allowlists enforced by the runtime.
type HandlerRef struct {
	File    string
	Sha256  string
	HostAPI int
	// Entry is the exported function name to invoke; empty falls back to "handle".
	Entry string
	// Calls is the URL-pattern allowlist for host.http.request.
	Calls []string
	// Credentials is the allowlist of keys host.creds.get may resolve.
	Credentials []string
	// FailCodes is the allowlist of codes host.fail.withCode may emit.
	FailCodes []string
	// MemoryMB raises the per-invocation memory cap (0 = default 64; max 256).
	MemoryMB int
	// CPUSeconds raises the per-invocation CPU cap (0 = default 30; max 120).
	CPUSeconds int
}

// RequestSpec is the HTTP template for a declarative action.
type RequestSpec struct {
	Method  string            // GET, POST, PUT, PATCH, DELETE
	Path    string            // supports {var} interpolation
	Headers map[string]string // values support interpolation
	Body    string            // "$inputs" → JSON of body-located inputs; "{tpl}" → interpolated; "" → no body
}

// PaginationSpec declares cursor pagination behavior for an action.
type PaginationSpec struct {
	Style           string // "cursor" only in v0.2
	RequestParam    string
	RequestLocation string // "query" by default
	ResponseToken   string
	ResponseItems   string // JSON path to the items slice (empty = top-level array)
	MaxPages        int
}

// ErrorSpec declares the runtime mapping for a specific HTTP status code.
type ErrorSpec struct {
	Code         string
	Hint         string
	InstallGuide string
	Retry        string
	MaxAttempts  int
}

// Action defines a callable operation on a service.
type Action struct {
	ID          ActionID
	Service     ServiceID
	Description string
	Permission  PermissionPath
	InputSchema json.RawMessage // serialized []InputDef; parse via ParseInputSchema
	Request     *RequestSpec    // nil iff Handler is set (WASM)
	Pagination  *PaginationSpec
	Errors      map[int]ErrorSpec // keyed by HTTP status code
	Handler     *HandlerRef
	SideEffects SideEffect
	// Source identifies where this action's definition was loaded from.
	// Empty means the built-in / official catalog. A non-empty value of the
	// form "tap:<name>" means the action comes from a third-party tap; the
	// runtime treats such actions as lower trust (e.g. WASM handlers refused
	// unless explicitly allowlisted).
	Source string
}

// SideEffect classifies an action by its impact. Defaults to read.
type SideEffect string

const (
	SideEffectRead        SideEffect = "read"
	SideEffectWrite       SideEffect = "write"
	SideEffectDestructive SideEffect = "destructive"
)

// IsDeclarative reports whether the action uses the built-in HTTP declarative runtime.
func (a Action) IsDeclarative() bool { return a.Handler == nil }

// IsDestructive reports whether the action is marked destructive in the catalog.
func (a Action) IsDestructive() bool { return a.SideEffects == SideEffectDestructive }
