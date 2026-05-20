package runtime

import (
	"fmt"
	"net/url"
	"strings"
)

// Interpolate substitutes `{name}` placeholders in tpl with values from vars.
// Returns an error if a placeholder has no matching var or if the substituted
// value would introduce path traversal (`..` or `/`).
//
// `urlEscape` controls whether values are URL-path-escaped before substitution.
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
		end := strings.IndexByte(tpl[i:], '}')
		if end < 0 {
			return "", fmt.Errorf("unclosed `{` at position %d", i)
		}
		name := tpl[i+1 : i+end]
		val, ok := vars[name]
		if !ok {
			return "", fmt.Errorf("unresolved placeholder {%s}", name)
		}
		s := fmt.Sprint(val)
		if urlEscape {
			if strings.Contains(s, "..") || strings.Contains(s, "/") {
				return "", fmt.Errorf("placeholder {%s} value %q would introduce path traversal", name, s)
			}
			s = url.PathEscape(s)
		}
		sb.WriteString(s)
		i += end + 1
	}
	return sb.String(), nil
}
