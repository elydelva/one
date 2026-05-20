package core

import "encoding/json"

// ActionID identifies an action within a service (e.g. "issues.create").
type ActionID string

// HandlerRef points to a WASM handler file in the catalog.
type HandlerRef struct {
	File    string
	Sha256  string
	HostAPI int
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
	Style          string // "cursor" only in v0.2
	RequestParam   string
	RequestLocation string // "query" by default
	ResponseToken  string
	ResponseItems  string // JSON path to the items slice (empty = top-level array)
	MaxPages       int
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
}

// IsDeclarative reports whether the action uses the built-in HTTP declarative runtime.
func (a Action) IsDeclarative() bool { return a.Handler == nil }
