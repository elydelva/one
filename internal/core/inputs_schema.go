package core

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// InputLocation indicates where an input parameter is sent in the HTTP request.
type InputLocation string

const (
	LocationPath   InputLocation = "path"
	LocationQuery  InputLocation = "query"
	LocationBody   InputLocation = "body"
	LocationHeader InputLocation = "header"
)

// InputType is the JSON Schema subset supported in v0.2.
type InputType string

const (
	TypeString  InputType = "string"
	TypeInteger InputType = "integer"
	TypeNumber  InputType = "number"
	TypeBoolean InputType = "boolean"
	TypeArray   InputType = "array"
	TypeObject  InputType = "object"
	TypeFileRef InputType = "file_ref"
)

// InputDef describes a single action parameter (parsed from the catalog YAML).
type InputDef struct {
	Name        string        `json:"name"`
	Type        InputType     `json:"type"`
	Required    bool          `json:"required,omitempty"`
	Location    InputLocation `json:"location,omitempty"` // default: path if name appears in path template, else query
	Description string        `json:"description,omitempty"`
	Pattern     string        `json:"pattern,omitempty"`
	Enum        []any         `json:"enum,omitempty"`
	Default     any           `json:"default,omitempty"`
	Min         *float64      `json:"min,omitempty"`
	Max         *float64      `json:"max,omitempty"`
	MinLen      *int          `json:"min_length,omitempty"`
	MaxLen      *int          `json:"max_length,omitempty"`
	Items       *InputDef     `json:"items,omitempty"` // when Type == array
}

// InputSchema is the validated schema for an action's inputs.
type InputSchema struct {
	Defs []InputDef
	// compiled regex per field name
	patterns map[string]*regexp.Regexp
}

// ParseInputSchema unmarshals an InputSchema from JSON and compiles patterns.
func ParseInputSchema(data json.RawMessage) (InputSchema, error) {
	if len(data) == 0 {
		return InputSchema{patterns: map[string]*regexp.Regexp{}}, nil
	}
	var defs []InputDef
	if err := json.Unmarshal(data, &defs); err != nil {
		return InputSchema{}, fmt.Errorf("parse input schema: %w", err)
	}
	out := InputSchema{Defs: defs, patterns: map[string]*regexp.Regexp{}}
	for _, d := range defs {
		if d.Pattern == "" {
			continue
		}
		re, err := regexp.Compile(d.Pattern)
		if err != nil {
			return InputSchema{}, fmt.Errorf("input %q: invalid pattern: %w", d.Name, err)
		}
		out.patterns[d.Name] = re
	}
	return out, nil
}

// Validate checks the provided inputs against the schema. Returns the first
// validation error encountered.
func (s InputSchema) Validate(inputs Inputs) error {
	for _, def := range s.Defs {
		v, present := inputs[def.Name]
		if !present {
			if def.Required && def.Default == nil {
				return ErrInputValidation{Field: def.Name, Reason: "required"}
			}
			continue
		}
		if err := s.validateValue(def, v); err != nil {
			return err
		}
	}
	return nil
}

func (s InputSchema) validateValue(def InputDef, v any) error {
	switch def.Type {
	case TypeString, TypeFileRef:
		str, ok := v.(string)
		if !ok {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("expected string, got %T", v)}
		}
		if def.MinLen != nil && len(str) < *def.MinLen {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("min_length %d", *def.MinLen)}
		}
		if def.MaxLen != nil && len(str) > *def.MaxLen {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("max_length %d", *def.MaxLen)}
		}
		if re := s.patterns[def.Name]; re != nil && !re.MatchString(str) {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("does not match pattern %q", def.Pattern)}
		}
		if len(def.Enum) > 0 && !inEnum(str, def.Enum) {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("not in enum %v", def.Enum)}
		}
	case TypeInteger:
		f, ok := toFloat(v)
		if !ok || f != float64(int64(f)) {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("expected integer, got %v", v)}
		}
		if err := numericBounds(def, f); err != nil {
			return err
		}
	case TypeNumber:
		f, ok := toFloat(v)
		if !ok {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("expected number, got %v", v)}
		}
		if err := numericBounds(def, f); err != nil {
			return err
		}
	case TypeBoolean:
		if _, ok := v.(bool); !ok {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("expected bool, got %T", v)}
		}
	case TypeArray:
		arr, ok := v.([]any)
		if !ok {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("expected array, got %T", v)}
		}
		if def.Items != nil {
			for i, item := range arr {
				itemDef := *def.Items
				itemDef.Name = fmt.Sprintf("%s[%d]", def.Name, i)
				if err := s.validateValue(itemDef, item); err != nil {
					return err
				}
			}
		}
	case TypeObject:
		if _, ok := v.(map[string]any); !ok {
			return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("expected object, got %T", v)}
		}
	default:
		return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("unsupported type %q", def.Type)}
	}
	return nil
}

func numericBounds(def InputDef, f float64) error {
	if def.Min != nil && f < *def.Min {
		return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("min %v", *def.Min)}
	}
	if def.Max != nil && f > *def.Max {
		return ErrInputValidation{Field: def.Name, Reason: fmt.Sprintf("max %v", *def.Max)}
	}
	return nil
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func inEnum(v any, enum []any) bool {
	for _, e := range enum {
		if e == v {
			return true
		}
	}
	return false
}

// ApplyDefaults returns a copy of inputs with defaults filled in for missing fields.
func (s InputSchema) ApplyDefaults(inputs Inputs) Inputs {
	out := Inputs{}
	for k, v := range inputs {
		out[k] = v
	}
	for _, def := range s.Defs {
		if _, ok := out[def.Name]; ok {
			continue
		}
		if def.Default != nil {
			out[def.Name] = def.Default
		}
	}
	return out
}

// ByLocation returns inputs grouped by their declared HTTP location.
func (s InputSchema) ByLocation(inputs Inputs) map[InputLocation]map[string]any {
	out := map[InputLocation]map[string]any{
		LocationPath:   {},
		LocationQuery:  {},
		LocationBody:   {},
		LocationHeader: {},
	}
	for _, def := range s.Defs {
		v, ok := inputs[def.Name]
		if !ok {
			continue
		}
		loc := def.Location
		if loc == "" {
			loc = LocationQuery
		}
		out[loc][def.Name] = v
	}
	return out
}
