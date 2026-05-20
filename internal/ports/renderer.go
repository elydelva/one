package ports

import "encoding/json"

// CapabilitiesOutput is the structured introspection payload for `one capabilities`.
type CapabilitiesOutput struct {
	Services []ServiceCapability `json:"services"`
}

// ServiceCapability describes what a service can do.
type ServiceCapability struct {
	ID      string   `json:"id"`
	Actions []string `json:"actions"`
}

// Renderer formats and writes output to stdout/stderr.
type Renderer interface {
	RenderResult(output json.RawMessage, traceID string)
	RenderError(err error)
	RenderInfo(markdown string)
	RenderCapabilities(out CapabilitiesOutput)
}
