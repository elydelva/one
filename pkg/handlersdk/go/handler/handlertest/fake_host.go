// Package handlertest provides an in-process fake host so handler logic can
// be exercised without compiling to WASM.
package handlertest

import (
	"encoding/json"

	"elydelva/one/pkg/handlersdk/go/handler"
)

// FakeHost captures the input envelope, the produced output, and any fail.
type FakeHost struct {
	Inputs map[string]any
	Output json.RawMessage
	Fail   *FailCall
}

type FailCall struct{ Code, Msg, Hint string }

// Install wires the fake host onto the SDK's native bridges.
func (f *FakeHost) Install() {
	handler.SetTestHooks(
		func() []byte {
			env := map[string]any{"inputs": f.Inputs}
			bs, _ := json.Marshal(env)
			return bs
		},
		func(b []byte) { f.Output = append(json.RawMessage(nil), b...) },
		func(code, msg, hint string) { f.Fail = &FailCall{Code: code, Msg: msg, Hint: hint} },
	)
}
