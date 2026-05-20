// Package handler is the Go SDK for writing One CLI WASM handlers.
//
// Build with tinygo:
//
//	tinygo build -o handler.wasm -target=wasi -no-debug ./main.go
//
// The compiled binary must:
//   - declare its exports (e.g. `//export handle`),
//   - call host.* primitives only via this package,
//   - return via Output/Fail.
package handler

import "encoding/json"

// Envelope is what the host passes to read_inputs.
type Envelope struct {
	Action  string         `json:"action"`
	Inputs  map[string]any `json:"inputs"`
	Context map[string]any `json:"context"`
}

// ReadInputs decodes the host envelope and unmarshals Inputs into v.
func ReadInputs(v any) error {
	raw := readInputs()
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	if v == nil {
		return nil
	}
	bs, _ := json.Marshal(env.Inputs)
	return json.Unmarshal(bs, v)
}

// Output marshals v as JSON and reports it to the host as the action result.
func Output(v any) {
	bs, _ := json.Marshal(v)
	writeOutput(bs)
}

// Fail terminates the handler with a structured error.
// Code must appear in service.yaml > handler.fail_codes.
func Fail(code, msg, hint string) {
	fail(code, msg, hint)
}

// HTTP performs a request through host.http (subject to the action's calls allowlist).
type HTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"-"`
}

type HTTPResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    []byte            `json:"-"`
}
