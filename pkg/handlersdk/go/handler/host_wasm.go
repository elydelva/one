//go:build wasm

package handler

import "unsafe"

//go:wasmimport one read_inputs
func host_read_inputs(ptr uint32, max uint32) uint32

//go:wasmimport one write_output
func host_write_output(ptr uint32, length uint32)

//go:wasmimport one fail
func host_fail(codePtr, codeLen, msgPtr, msgLen, hintPtr, hintLen uint32)

const ioBufSize = 1 << 20 // 1 MiB

var ioBuf [ioBufSize]byte

func readInputs() []byte {
	ptr := uint32(uintptr(unsafe.Pointer(&ioBuf[0])))
	n := host_read_inputs(ptr, ioBufSize)
	if n == ^uint32(0) {
		return nil
	}
	out := make([]byte, n)
	copy(out, ioBuf[:n])
	return out
}

func writeOutput(b []byte) {
	if len(b) > ioBufSize {
		return
	}
	copy(ioBuf[:], b)
	host_write_output(uint32(uintptr(unsafe.Pointer(&ioBuf[0]))), uint32(len(b)))
}

func fail(code, msg, hint string) {
	cp := []byte(code)
	mp := []byte(msg)
	hp := []byte(hint)
	cPtr, cLen := stringPtr(cp)
	mPtr, mLen := stringPtr(mp)
	hPtr, hLen := stringPtr(hp)
	host_fail(cPtr, cLen, mPtr, mLen, hPtr, hLen)
}

func stringPtr(b []byte) (uint32, uint32) {
	if len(b) == 0 {
		return 0, 0
	}
	return uint32(uintptr(unsafe.Pointer(&b[0]))), uint32(len(b))
}
