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

// Action defines a callable operation on a service.
type Action struct {
	ID          ActionID
	Service     ServiceID
	Description string
	Permission  PermissionPath
	InputSchema json.RawMessage
	Handler     *HandlerRef
}

// IsDeclarative reports whether the action uses the built-in HTTP declarative runtime.
func (a Action) IsDeclarative() bool { return a.Handler == nil }
