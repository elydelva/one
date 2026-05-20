//go:build !wasm

package handler

// Native build of the SDK: host bridges are stubbed. Used for unit tests in
// handler/handlertest (where a fake host overrides these functions via a hook).
var (
	readInputsHook  func() []byte
	writeOutputHook func([]byte)
	failHook        func(code, msg, hint string)
)

func readInputs() []byte {
	if readInputsHook != nil {
		return readInputsHook()
	}
	return []byte(`{"inputs":{}}`)
}

func writeOutput(b []byte) {
	if writeOutputHook != nil {
		writeOutputHook(b)
	}
}

func fail(code, msg, hint string) {
	if failHook != nil {
		failHook(code, msg, hint)
	}
}

// SetTestHooks replaces native bridges for in-process tests.
// Call only from handlertest.
func SetTestHooks(read func() []byte, write func([]byte), fl func(code, msg, hint string)) {
	readInputsHook = read
	writeOutputHook = write
	failHook = fl
}
