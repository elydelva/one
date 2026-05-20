//go:build !nowasm

package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// BenchmarkWazeroColdStart measures the per-invocation overhead of compiling
// and instantiating a trivial handler. CI threshold (PR18): p99 < 80 ms,
// captured in .benchmarks.json.
func BenchmarkWazeroColdStart(b *testing.B) {
	sum := sha256.Sum256(tinyHandleWasm)
	sha := hex.EncodeToString(sum[:])
	r := NewWazeroRuntime(nil, nil, fakeClock{}, discardLogger{}, &memResolver{bin: tinyHandleWasm}, b.TempDir())
	req := ports.ExecuteRequest{
		Action: core.Action{
			ID:      "bench.handle",
			Service: "bench",
			Handler: &core.HandlerRef{File: "h.wasm", Sha256: sha, HostAPI: HostAPIVersion, Entry: "handle"},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.Execute(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}
