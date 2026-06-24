package catalog

import (
	"fmt"
	"regexp"
)

// LintFinding describes a static issue detected in a handler's source code or
// its service.yaml declaration.
type LintFinding struct {
	Service string
	Action  string
	Kind    string // "url_not_in_calls", "cred_not_declared", "fail_code_not_declared", "shape"
	Detail  string
}

func (f LintFinding) Error() string {
	return fmt.Sprintf("%s.%s [%s]: %s", f.Service, f.Action, f.Kind, f.Detail)
}

// LintHandlerSource inspects a TS or Go handler source for calls/credentials/codes
// references and verifies each lives in the action's declared allowlists.
// This is a best-effort regex scan — runtime allowlists remain the source of truth.
func LintHandlerSource(svc string, action string, def ActionDef, source string) []LintFinding {
	if def.Handler == nil {
		return nil
	}
	var findings []LintFinding
	findings = append(findings, checkURLs(svc, action, def.Handler.Calls, source)...)
	findings = append(findings, checkCreds(svc, action, def.Handler.Credentials, source)...)
	findings = append(findings, checkFails(svc, action, def.Handler.FailCodes, source)...)
	return findings
}

var (
	reURL  = regexp.MustCompile(`(?:["'])(https?://[^"'\s]+)(?:["'])`)
	reCred = regexp.MustCompile(`(?:host\.creds\.get|creds_get)\s*\(\s*["']([^"']+)["']`)
	reFail = regexp.MustCompile(`(?:host\.fail(?:\.withCode)?|fail)\s*\(\s*["']([^"']+)["']`)
)

func checkURLs(svc, action string, allowed []string, source string) []LintFinding {
	patterns := compilePatterns(allowed)
	var out []LintFinding
	for _, m := range reURL.FindAllStringSubmatch(source, -1) {
		url := m[1]
		matched := false
		for _, p := range patterns {
			if p.MatchString(url) {
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, LintFinding{Service: svc, Action: action, Kind: "url_not_in_calls", Detail: url})
		}
	}
	return out
}

func checkCreds(svc, action string, allowed []string, source string) []LintFinding {
	set := toSet(allowed)
	var out []LintFinding
	for _, m := range reCred.FindAllStringSubmatch(source, -1) {
		if _, ok := set[m[1]]; !ok {
			out = append(out, LintFinding{Service: svc, Action: action, Kind: "cred_not_declared", Detail: m[1]})
		}
	}
	return out
}

func checkFails(svc, action string, allowed []string, source string) []LintFinding {
	set := toSet(allowed)
	var out []LintFinding
	for _, m := range reFail.FindAllStringSubmatch(source, -1) {
		if _, ok := set[m[1]]; !ok {
			out = append(out, LintFinding{Service: svc, Action: action, Kind: "fail_code_not_declared", Detail: m[1]})
		}
	}
	return out
}

func compilePatterns(in []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(in))
	for _, p := range in {
		if p == "" {
			continue
		}
		anchored := p
		if anchored[0] != '^' {
			anchored = "^" + anchored
		}
		if anchored[len(anchored)-1] != '$' {
			anchored += "$"
		}
		if re, err := regexp.Compile(anchored); err == nil {
			out = append(out, re)
		}
	}
	return out
}

func toSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, s := range in {
		out[s] = struct{}{}
	}
	return out
}
