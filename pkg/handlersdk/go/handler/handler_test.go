package handler_test

import (
	"encoding/json"
	"testing"

	"elydelva/one/pkg/handlersdk/go/handler"
	"elydelva/one/pkg/handlersdk/go/handler/handlertest"
)

func TestEchoHandlerRoundTrip(t *testing.T) {
	fh := &handlertest.FakeHost{Inputs: map[string]any{"msg": "hello"}}
	fh.Install()

	var in struct {
		Msg string `json:"msg"`
	}
	if err := handler.ReadInputs(&in); err != nil {
		t.Fatal(err)
	}
	if in.Msg != "hello" {
		t.Fatalf("want hello, got %q", in.Msg)
	}
	handler.Output(map[string]string{"echo": in.Msg})

	var got map[string]string
	if err := json.Unmarshal(fh.Output, &got); err != nil {
		t.Fatal(err)
	}
	if got["echo"] != "hello" {
		t.Fatalf("want echo=hello, got %v", got)
	}
}

func TestFailCapturedByFakeHost(t *testing.T) {
	fh := &handlertest.FakeHost{}
	fh.Install()
	handler.Fail("not_found", "missing", "one install x")
	if fh.Fail == nil || fh.Fail.Code != "not_found" {
		t.Fatalf("fail not captured: %+v", fh.Fail)
	}
}
