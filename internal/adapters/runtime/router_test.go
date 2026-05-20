package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/fake"
)

func TestRouter_DispatchesDeclarative(t *testing.T) {
	declarative := fake.NewRuntime()
	declarative.SetResponse("svc", "act", json.RawMessage(`{"ok":true}`))
	r := NewRouter(declarative, nil)
	res, err := r.Execute(context.Background(), ports.ExecuteRequest{
		Action: core.Action{ID: "act", Service: "svc"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if string(res.Output) != `{"ok":true}` {
		t.Errorf("got %s", res.Output)
	}
}

func TestRouter_RefusesWASMWhenNil(t *testing.T) {
	r := NewRouter(fake.NewRuntime(), nil)
	_, err := r.Execute(context.Background(), ports.ExecuteRequest{
		Action: core.Action{ID: "act", Service: "svc", Handler: &core.HandlerRef{File: "x.wasm"}},
	})
	var unsup core.ErrUnsupportedRuntime
	if !errors.As(err, &unsup) {
		t.Errorf("got %T: %v", err, err)
	}
}
