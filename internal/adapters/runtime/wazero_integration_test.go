//go:build !nowasm

package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// Minimal valid WASM module: `(module (func (export "handle")))`. Exports
// a single no-op function named handle.
var tinyHandleWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
	// type section: 1 func type () -> ()
	0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
	// function section: 1 function of type 0
	0x03, 0x02, 0x01, 0x00,
	// export section: 1 export "handle" func 0
	0x07, 0x0a, 0x01, 0x06, 'h', 'a', 'n', 'd', 'l', 'e', 0x00, 0x00,
	// code section: 1 body (size=2, locals=0, end)
	0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b,
}

type memResolver struct{ bin []byte }

func (m *memResolver) ReadHandler(_ context.Context, _ core.ServiceID, _ string) ([]byte, error) {
	return m.bin, nil
}

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Unix(1700000000, 0) }

type discardLogger struct{}

func (d discardLogger) Debug(string, ...any)     {}
func (d discardLogger) Info(string, ...any)      {}
func (d discardLogger) Warn(string, ...any)      {}
func (d discardLogger) Error(string, ...any)     {}
func (d discardLogger) With(...any) ports.Logger { return d }

func TestWazeroRunsTinyHandler(t *testing.T) {
	sum := sha256.Sum256(tinyHandleWasm)
	sha := hex.EncodeToString(sum[:])

	r := NewWazeroRuntime(nil, nil, fakeClock{}, discardLogger{}, &memResolver{bin: tinyHandleWasm}, "")
	req := ports.ExecuteRequest{
		Action: core.Action{
			ID:      "test.handle",
			Service: "test",
			Handler: &core.HandlerRef{File: "h.wasm", Sha256: sha, HostAPI: HostAPIVersion, Entry: "handle"},
		},
	}
	res, err := r.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(res.Output) != "null" {
		t.Fatalf("expected null output, got %q", res.Output)
	}
}

func TestWazeroRejectsBadSHA256(t *testing.T) {
	r := NewWazeroRuntime(nil, nil, fakeClock{}, discardLogger{}, &memResolver{bin: tinyHandleWasm}, "")
	req := ports.ExecuteRequest{
		Action: core.Action{
			ID:      "test.handle",
			Service: "test",
			Handler: &core.HandlerRef{File: "h.wasm", Sha256: "deadbeef", HostAPI: HostAPIVersion, Entry: "handle"},
		},
	}
	_, err := r.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("expected sha256 violation")
	}
}

func TestWazeroAOTCachePersists(t *testing.T) {
	dir := t.TempDir()
	sum := sha256.Sum256(tinyHandleWasm)
	sha := hex.EncodeToString(sum[:])
	r := NewWazeroRuntime(nil, nil, fakeClock{}, discardLogger{}, &memResolver{bin: tinyHandleWasm}, dir)
	req := ports.ExecuteRequest{
		Action: core.Action{
			ID:      "test.handle",
			Service: "test",
			Handler: &core.HandlerRef{File: "h.wasm", Sha256: sha, HostAPI: HostAPIVersion, Entry: "handle"},
		},
	}
	if _, err := r.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	// second call should hit the in-memory cache (no error)
	if _, err := r.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
}
