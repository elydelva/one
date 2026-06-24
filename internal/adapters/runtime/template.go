package runtime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Interpolate substitutes `{name}` and `{name|filter}` placeholders in tpl with
// values from vars. A `{` followed by anything other than a valid identifier
// character is emitted literally — this makes JSON body templates safe to write
// (e.g. `{"content":{c|json}}` parses one placeholder `{c|json}`).
//
// Filters:
//   - (none)  : fmt.Sprint(value)
//   - json    : json.Marshal(value) — yields a JSON literal (quoted strings,
//     escaped specials, booleans/numbers/objects unquoted). Use this
//     to safely embed values inside JSON body templates.
//
// urlEscape applies after filtering. With urlEscape=true, the substituted
// string must not contain `..` or `/` (anti path-traversal).
func Interpolate(tpl string, vars map[string]any, urlEscape bool) (string, error) {
	var sb strings.Builder
	i := 0
	for i < len(tpl) {
		c := tpl[i]
		if c != '{' {
			sb.WriteByte(c)
			i++
			continue
		}
		// Probe a valid placeholder: {<name>(|<filter>)?}.
		end := findPlaceholderEnd(tpl, i)
		if end < 0 {
			// Not a placeholder — emit the `{` literally and continue.
			sb.WriteByte('{')
			i++
			continue
		}
		expr := tpl[i+1 : end]
		name, filter, _ := strings.Cut(expr, "|")
		val, ok := vars[name]
		if !ok {
			return "", fmt.Errorf("unresolved placeholder {%s}", name)
		}
		var s string
		switch filter {
		case "":
			s = fmt.Sprint(val)
		case "json":
			b, err := json.Marshal(val)
			if err != nil {
				return "", fmt.Errorf("placeholder {%s|json}: %w", name, err)
			}
			s = string(b)
		default:
			return "", fmt.Errorf("placeholder {%s}: unknown filter %q", name, filter)
		}
		if urlEscape {
			if strings.Contains(s, "..") || strings.Contains(s, "/") {
				return "", fmt.Errorf("placeholder {%s} value %q would introduce path traversal", name, s)
			}
			s = url.PathEscape(s)
		}
		sb.WriteString(s)
		i = end + 1
	}
	return sb.String(), nil
}

// findPlaceholderEnd returns the index of the closing `}` for a placeholder
// starting at tpl[start] == '{', or -1 if the run is not a valid placeholder.
// Valid: `{` <ident> (`|` <ident>)? `}` where ident = [A-Za-z_][A-Za-z0-9_]*.
func findPlaceholderEnd(tpl string, start int) int {
	i := start + 1
	if i >= len(tpl) || !isIdentStart(tpl[i]) {
		return -1
	}
	for i < len(tpl) && isIdentPart(tpl[i]) {
		i++
	}
	if i < len(tpl) && tpl[i] == '|' {
		i++
		if i >= len(tpl) || !isIdentStart(tpl[i]) {
			return -1
		}
		for i < len(tpl) && isIdentPart(tpl[i]) {
			i++
		}
	}
	if i >= len(tpl) || tpl[i] != '}' {
		return -1
	}
	return i
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
