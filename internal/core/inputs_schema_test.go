package core

import (
	"encoding/json"
	"errors"
	"testing"
)

func parse(t *testing.T, s string) InputSchema {
	t.Helper()
	sch, err := ParseInputSchema(json.RawMessage(s))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return sch
}

func TestValidate_RequiredMissing(t *testing.T) {
	s := parse(t, `[{"name":"repo","type":"string","required":true}]`)
	err := s.Validate(Inputs{})
	var iv ErrInputValidation
	if !errors.As(err, &iv) || iv.Field != "repo" {
		t.Errorf("want ErrInputValidation{repo}, got %v", err)
	}
}

func TestValidate_OptionalAbsent(t *testing.T) {
	s := parse(t, `[{"name":"state","type":"string"}]`)
	if err := s.Validate(Inputs{}); err != nil {
		t.Errorf("optional absent should be OK, got %v", err)
	}
}

func TestValidate_StringPattern(t *testing.T) {
	s := parse(t, `[{"name":"slug","type":"string","pattern":"^[a-z]+$"}]`)
	if err := s.Validate(Inputs{"slug": "abc"}); err != nil {
		t.Errorf("ok value rejected: %v", err)
	}
	if err := s.Validate(Inputs{"slug": "ABC123"}); err == nil {
		t.Errorf("bad value accepted")
	}
}

func TestValidate_Enum(t *testing.T) {
	s := parse(t, `[{"name":"state","type":"string","enum":["open","closed"]}]`)
	if err := s.Validate(Inputs{"state": "open"}); err != nil {
		t.Errorf("open rejected: %v", err)
	}
	if err := s.Validate(Inputs{"state": "draft"}); err == nil {
		t.Errorf("draft accepted")
	}
}

func TestValidate_IntegerStrict(t *testing.T) {
	s := parse(t, `[{"name":"n","type":"integer"}]`)
	if err := s.Validate(Inputs{"n": 5}); err != nil {
		t.Errorf("int rejected: %v", err)
	}
	if err := s.Validate(Inputs{"n": 5.5}); err == nil {
		t.Errorf("float accepted as integer")
	}
}

func TestValidate_NumericBounds(t *testing.T) {
	s := parse(t, `[{"name":"n","type":"integer","min":1,"max":100}]`)
	if err := s.Validate(Inputs{"n": 0}); err == nil {
		t.Errorf("below min accepted")
	}
	if err := s.Validate(Inputs{"n": 101}); err == nil {
		t.Errorf("above max accepted")
	}
	if err := s.Validate(Inputs{"n": 50}); err != nil {
		t.Errorf("in range rejected: %v", err)
	}
}

func TestValidate_StringLength(t *testing.T) {
	s := parse(t, `[{"name":"s","type":"string","min_length":2,"max_length":4}]`)
	if err := s.Validate(Inputs{"s": "a"}); err == nil {
		t.Errorf("too short accepted")
	}
	if err := s.Validate(Inputs{"s": "abcde"}); err == nil {
		t.Errorf("too long accepted")
	}
	if err := s.Validate(Inputs{"s": "abc"}); err != nil {
		t.Errorf("ok rejected: %v", err)
	}
}

func TestValidate_Boolean(t *testing.T) {
	s := parse(t, `[{"name":"b","type":"boolean"}]`)
	if err := s.Validate(Inputs{"b": true}); err != nil {
		t.Errorf("bool rejected: %v", err)
	}
	if err := s.Validate(Inputs{"b": "true"}); err == nil {
		t.Errorf("string accepted as bool")
	}
}

func TestValidate_ArrayItems(t *testing.T) {
	s := parse(t, `[{"name":"labels","type":"array","items":{"name":"_","type":"string"}}]`)
	if err := s.Validate(Inputs{"labels": []any{"a", "b"}}); err != nil {
		t.Errorf("good array rejected: %v", err)
	}
	if err := s.Validate(Inputs{"labels": []any{"a", 5}}); err == nil {
		t.Errorf("array with bad item accepted")
	}
	if err := s.Validate(Inputs{"labels": "not-array"}); err == nil {
		t.Errorf("non-array accepted")
	}
}

func TestValidate_Object(t *testing.T) {
	s := parse(t, `[{"name":"meta","type":"object"}]`)
	if err := s.Validate(Inputs{"meta": map[string]any{"k": "v"}}); err != nil {
		t.Errorf("object rejected: %v", err)
	}
	if err := s.Validate(Inputs{"meta": "string"}); err == nil {
		t.Errorf("string accepted as object")
	}
}

func TestValidate_FileRefIsString(t *testing.T) {
	s := parse(t, `[{"name":"body","type":"file_ref"}]`)
	if err := s.Validate(Inputs{"body": "expanded-content"}); err != nil {
		t.Errorf("file_ref string rejected: %v", err)
	}
}

func TestApplyDefaults(t *testing.T) {
	s := parse(t, `[{"name":"page","type":"integer","default":1}]`)
	out := s.ApplyDefaults(Inputs{})
	if out["page"] != float64(1) && out["page"] != 1 {
		t.Errorf("default not applied: %+v", out)
	}
}

func TestByLocation(t *testing.T) {
	s := parse(t, `[
		{"name":"owner","type":"string","location":"path"},
		{"name":"state","type":"string","location":"query"},
		{"name":"title","type":"string","location":"body"}
	]`)
	groups := s.ByLocation(Inputs{"owner": "x", "state": "open", "title": "t"})
	if groups[LocationPath]["owner"] != "x" {
		t.Errorf("path group missing owner: %+v", groups)
	}
	if groups[LocationQuery]["state"] != "open" {
		t.Errorf("query group missing state: %+v", groups)
	}
	if groups[LocationBody]["title"] != "t" {
		t.Errorf("body group missing title: %+v", groups)
	}
}

func TestParseInputSchema_InvalidPattern(t *testing.T) {
	_, err := ParseInputSchema(json.RawMessage(`[{"name":"x","type":"string","pattern":"["}]`))
	if err == nil {
		t.Errorf("expected error on invalid regex")
	}
}

func TestParseInputSchema_Empty(t *testing.T) {
	s, err := ParseInputSchema(nil)
	if err != nil {
		t.Errorf("nil should parse: %v", err)
	}
	if err := s.Validate(Inputs{}); err != nil {
		t.Errorf("empty schema should accept anything: %v", err)
	}
}
