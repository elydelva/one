package main

import (
	"context"
	"os/exec"
	"testing"
)

// BenchmarkColdStart_Version invokes the built binary with --version. This
// captures fork+exec + Go runtime init + cobra setup, which is what users
// actually pay on every CLI invocation.
//
// Budget enforced via .benchmarks.json: max_ns_per_op = 30 ms.
//
// The binary must be built ahead of time (CI: `make build` or `go build -o
// bin/one ./cmd/one`); the test skips if missing.
func BenchmarkColdStart_Version(b *testing.B) {
	if _, err := exec.LookPath("bin/one"); err != nil {
		b.Skip("bin/one not built — run `make build` first")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.CommandContext(context.Background(), "bin/one", "--version")
		if err := cmd.Run(); err != nil {
			b.Fatal(err)
		}
	}
}
