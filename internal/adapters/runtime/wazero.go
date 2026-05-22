//go:build !nowasm

package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// HostAPIVersion is the host ABI version. Bumped on any breaking host-fn change.
const HostAPIVersion = 1

// Per-invocation caps. Catalog may raise within MaxMemoryMB / MaxCPUSeconds.
const (
	DefaultMemoryMB   = 64
	MaxMemoryMB       = 256
	DefaultCPUSeconds = 30
	MaxCPUSeconds     = 120
	MaxHTTPCalls      = 50
	MaxLogLines       = 1000
	MaxSleepPerCallMS = 30_000
	MaxSleepTotalMS   = 60_000
)

// HandlerResolver reads a handler binary referenced by an action.
type HandlerResolver interface {
	ReadHandler(ctx context.Context, svc core.ServiceID, file string) ([]byte, error)
}

// WazeroRuntime executes WASM handlers in a sandboxed wazero environment.
type WazeroRuntime struct {
	transport ports.Transport
	vault     ports.Vault
	clock     ports.Clock
	log       ports.Logger
	resolver  HandlerResolver

	cacheDir string
	cache    wazero.CompilationCache
}

// NewWazeroRuntime creates a WASM runtime. cacheDir may be "" to disable the
// on-disk compilation cache.
func NewWazeroRuntime(transport ports.Transport, vault ports.Vault, clock ports.Clock, log ports.Logger, resolver HandlerResolver, cacheDir string) *WazeroRuntime {
	r := &WazeroRuntime{
		transport: transport,
		vault:     vault,
		clock:     clock,
		log:       log,
		resolver:  resolver,
		cacheDir:  cacheDir,
	}
	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0o700); err == nil {
			if c, err := wazero.NewCompilationCacheWithDir(cacheDir); err == nil {
				r.cache = c
			}
		}
	}
	return r
}

var _ ports.Runtime = (*WazeroRuntime)(nil)

// Execute runs the WASM handler for the given action.
func (r *WazeroRuntime) Execute(ctx context.Context, req ports.ExecuteRequest) (ports.ExecuteResult, error) {
	h := req.Action.Handler
	if h == nil {
		return ports.ExecuteResult{}, fmt.Errorf("wazero: action %s has no handler", req.Action.ID)
	}
	if err := checkSourceTrust(req.Action); err != nil {
		return ports.ExecuteResult{}, err
	}
	if h.HostAPI != 0 && h.HostAPI != HostAPIVersion {
		return ports.ExecuteResult{}, core.ErrSandboxViolation{
			Action: req.Action.ID,
			Kind:   "host_api_version",
			Detail: fmt.Sprintf("handler requires v%d, host provides v%d", h.HostAPI, HostAPIVersion),
		}
	}

	memMB := clampInt(h.MemoryMB, DefaultMemoryMB, 1, MaxMemoryMB)
	cpuS := clampInt(h.CPUSeconds, DefaultCPUSeconds, 1, MaxCPUSeconds)

	bin, err := r.loadHandler(ctx, req.Action.Service, h)
	if err != nil {
		return ports.ExecuteResult{}, err
	}

	pages := uint32(memMB * 16) // 64 KiB per page → 16 pages/MB
	cfg := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(pages).
		WithCloseOnContextDone(true)
	if r.cache != nil {
		cfg = cfg.WithCompilationCache(r.cache)
	}
	rt := wazero.NewRuntimeWithConfig(ctx, cfg)
	defer rt.Close(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		return ports.ExecuteResult{}, fmt.Errorf("wazero: wasi: %w", err)
	}

	st := newInvocationState(req, r)
	if err := r.registerHostModule(ctx, rt, st); err != nil {
		return ports.ExecuteResult{}, err
	}

	compiled, err := r.compile(ctx, rt, bin, h.Sha256)
	if err != nil {
		return ports.ExecuteResult{}, fmt.Errorf("wazero: compile: %w", err)
	}

	cpuCtx, cancel := context.WithTimeout(ctx, time.Duration(cpuS)*time.Second)
	defer cancel()

	modCfg := wazero.NewModuleConfig().
		WithStdout(io.Discard).
		WithStderr(io.Discard).
		WithName("")

	mod, err := rt.InstantiateModule(cpuCtx, compiled, modCfg)
	if err != nil {
		return ports.ExecuteResult{}, r.classifyExecError(req.Action.ID, st, cpuCtx, cpuS, err)
	}
	defer mod.Close(cpuCtx)

	entry := h.Entry
	if entry == "" {
		entry = "handle"
	}
	fn := mod.ExportedFunction(entry)
	if fn != nil {
		if _, err := fn.Call(cpuCtx); err != nil {
			return ports.ExecuteResult{}, r.classifyExecError(req.Action.ID, st, cpuCtx, cpuS, err)
		}
	} else if st.output == nil && st.failure == nil {
		return ports.ExecuteResult{}, fmt.Errorf("wazero: handler exports neither %q nor _start with output", entry)
	}

	if st.failure != nil {
		return ports.ExecuteResult{}, *st.failure
	}
	if v, ok := st.violation(); ok {
		return ports.ExecuteResult{}, v
	}
	if st.output == nil {
		st.output = json.RawMessage(`null`)
	}
	return ports.ExecuteResult{Output: st.output, Calls: st.httpCalls}, nil
}

func (r *WazeroRuntime) classifyExecError(action core.ActionID, st *invocationState, ctx context.Context, cpuS int, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return core.ErrResourceExhausted{Action: action, Resource: "cpu", Cap: fmt.Sprintf("%ds", cpuS)}
	}
	if st.failure != nil {
		return *st.failure
	}
	if v, ok := st.violation(); ok {
		return v
	}
	return fmt.Errorf("wazero: invoke: %w", err)
}

func (r *WazeroRuntime) loadHandler(ctx context.Context, svc core.ServiceID, h *core.HandlerRef) ([]byte, error) {
	bin, err := r.resolver.ReadHandler(ctx, svc, h.File)
	if err != nil {
		return nil, fmt.Errorf("wazero: load handler %s: %w", h.File, err)
	}
	if h.Sha256 != "" {
		sum := sha256.Sum256(bin)
		if got := hex.EncodeToString(sum[:]); got != h.Sha256 {
			return nil, core.ErrSandboxViolation{
				Kind: "sha256_mismatch", Detail: fmt.Sprintf("expected %s got %s", h.Sha256, got),
			}
		}
	}
	return bin, nil
}

func (r *WazeroRuntime) compile(ctx context.Context, rt wazero.Runtime, bin []byte, _ string) (wazero.CompiledModule, error) {
	return rt.CompileModule(ctx, bin)
}

type invocationState struct {
	req     ports.ExecuteRequest
	rt      *WazeroRuntime
	inputs  []byte
	output  json.RawMessage
	failure *core.ErrHandlerFailed

	httpCalls []ports.HTTPCall
	httpCount int32
	logCount  int32
	sleepMS   int64

	mu    sync.Mutex
	viol  *core.ErrSandboxViolation
	resEx *core.ErrResourceExhausted
}

func newInvocationState(req ports.ExecuteRequest, rt *WazeroRuntime) *invocationState {
	envelope, _ := json.Marshal(struct {
		Action  string         `json:"action"`
		Inputs  core.Inputs    `json:"inputs"`
		Context map[string]any `json:"context"`
	}{
		Action: string(req.Action.ID),
		Inputs: req.Inputs,
		Context: map[string]any{
			"trace_id": req.TraceID,
			"dry_run":  req.DryRun,
		},
	})
	return &invocationState{req: req, rt: rt, inputs: envelope}
}

func (s *invocationState) setViolation(kind, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.viol == nil {
		s.viol = &core.ErrSandboxViolation{Action: s.req.Action.ID, Kind: kind, Detail: detail}
	}
}

func (s *invocationState) setResourceExhausted(resource, cap string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resEx == nil {
		s.resEx = &core.ErrResourceExhausted{Action: s.req.Action.ID, Resource: resource, Cap: cap}
	}
}

func (s *invocationState) violation() (error, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resEx != nil {
		return *s.resEx, true
	}
	if s.viol != nil {
		return *s.viol, true
	}
	return nil, false
}

func (r *WazeroRuntime) registerHostModule(ctx context.Context, rt wazero.Runtime, st *invocationState) error {
	b := rt.NewHostModuleBuilder("one")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, outPtr, outMax uint32) uint32 {
		if uint32(len(st.inputs)) > outMax {
			return ^uint32(0)
		}
		if !m.Memory().Write(outPtr, st.inputs) {
			return ^uint32(0)
		}
		return uint32(len(st.inputs))
	}).Export("read_inputs")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, ptr, length uint32) {
		buf, ok := m.Memory().Read(ptr, length)
		if !ok {
			st.setViolation("memory", "write_output out-of-bounds")
			return
		}
		cp := make([]byte, len(buf))
		copy(cp, buf)
		st.output = json.RawMessage(cp)
	}).Export("write_output")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, keyPtr, keyLen, outPtr, outMax uint32) uint32 {
		key, ok := readString(m, keyPtr, keyLen)
		if !ok {
			st.setViolation("memory", "creds_get key oob")
			return ^uint32(0)
		}
		if !contains(st.req.Action.Handler.Credentials, key) {
			st.setViolation("creds_not_declared", key)
			return ^uint32(0)
		}
		val, ok := credentialValue(st.req.Credential, key)
		if !ok {
			st.setViolation("creds_missing", key)
			return ^uint32(0)
		}
		bs := []byte(val)
		if uint32(len(bs)) > outMax {
			return ^uint32(0)
		}
		m.Memory().Write(outPtr, bs)
		return uint32(len(bs))
	}).Export("creds_get")

	b.NewFunctionBuilder().WithFunc(func(cctx context.Context, m api.Module, reqPtr, reqLen, outPtr, outMax uint32) uint32 {
		raw, ok := m.Memory().Read(reqPtr, reqLen)
		if !ok {
			st.setViolation("memory", "http_request oob")
			return ^uint32(0)
		}
		if atomic.AddInt32(&st.httpCount, 1) > MaxHTTPCalls {
			st.setResourceExhausted("http_calls", fmt.Sprintf("%d", MaxHTTPCalls))
			return ^uint32(0)
		}
		bs, err := r.doHostHTTP(cctx, st, raw)
		if err != nil {
			// Either captured as violation (URL not allowed) or surfaced inline.
			bs, _ = json.Marshal(map[string]any{"status": 0, "error": err.Error()})
		}
		if uint32(len(bs)) > outMax {
			return ^uint32(0)
		}
		m.Memory().Write(outPtr, bs)
		return uint32(len(bs))
	}).Export("http_request")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, levelPtr, levelLen, msgPtr, msgLen uint32) {
		if atomic.AddInt32(&st.logCount, 1) > MaxLogLines {
			return
		}
		level, _ := readString(m, levelPtr, levelLen)
		msg, _ := readString(m, msgPtr, msgLen)
		lg := r.log.With("handler", string(st.req.Action.ID), "trace_id", st.req.TraceID)
		switch level {
		case "debug":
			lg.Debug(msg)
		case "warn":
			lg.Warn(msg)
		default:
			lg.Info(msg)
		}
	}).Export("log_write")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, codePtr, codeLen, msgPtr, msgLen, hintPtr, hintLen uint32) {
		code, _ := readString(m, codePtr, codeLen)
		msg, _ := readString(m, msgPtr, msgLen)
		hint, _ := readString(m, hintPtr, hintLen)
		if !contains(st.req.Action.Handler.FailCodes, code) {
			st.setViolation("fail_code_not_declared", code)
			code = "unknown_error"
		}
		st.failure = &core.ErrHandlerFailed{Action: st.req.Action.ID, Code: code, Msg: msg, Hint: hint}
	}).Export("fail")

	b.NewFunctionBuilder().WithFunc(func(_ context.Context, _ api.Module) int64 {
		return r.clock.Now().UnixMilli()
	}).Export("time_now")

	b.NewFunctionBuilder().WithFunc(func(cctx context.Context, _ api.Module, ms uint32) {
		if ms > MaxSleepPerCallMS {
			st.setResourceExhausted("sleep_per_call", fmt.Sprintf("%dms", MaxSleepPerCallMS))
			return
		}
		if atomic.AddInt64(&st.sleepMS, int64(ms)) > MaxSleepTotalMS {
			st.setResourceExhausted("sleep_total", fmt.Sprintf("%dms", MaxSleepTotalMS))
			return
		}
		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
		case <-cctx.Done():
		}
	}).Export("time_sleep")

	registerCrypto(b)

	if _, err := b.Instantiate(ctx); err != nil {
		return fmt.Errorf("wazero: host module: %w", err)
	}
	return nil
}

func readString(m api.Module, ptr, length uint32) (string, bool) {
	if length == 0 {
		return "", true
	}
	buf, ok := m.Memory().Read(ptr, length)
	if !ok {
		return "", false
	}
	return string(buf), true
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func clampInt(v, def, lo, hi int) int {
	if v <= 0 {
		return def
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// credentialValue extracts a logical key from a core.Credential. Supports
// "access_token" / "refresh_token" / "api_key" (aliased to access token).
func credentialValue(c core.Credential, key string) (string, bool) {
	switch key {
	case "access_token", "api_key":
		v := c.AccessToken.Reveal()
		return v, v != ""
	case "refresh_token":
		v := c.RefreshToken.Reveal()
		return v, v != ""
	}
	return "", false
}
