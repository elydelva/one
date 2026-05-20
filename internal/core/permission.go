package core

import "strings"

// PermissionPath is the dot-separated path of a permission (e.g. "issues.read").
type PermissionPath string

// PermissionPattern is a glob pattern for matching permissions (e.g. "issues.*").
type PermissionPattern string

// Permission is a concrete grant: a service + path pair.
type Permission struct {
	Service ServiceID
	Path    PermissionPath
}

// Matches reports whether the pattern covers this permission.
// Supports simple wildcard (*) and exact match.
func (p PermissionPattern) Matches(perm Permission) bool {
	pattern := string(p)
	path := string(perm.Path)

	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == path
	}

	prefix := strings.TrimSuffix(pattern, "*")
	return strings.HasPrefix(path, prefix)
}

// ParsePermissionPattern validates a raw string and returns a PermissionPattern.
// Refuses `**`, `?`, brace expansion, and empty patterns.
func ParsePermissionPattern(s string) (PermissionPattern, error) {
	if s == "" {
		return "", ErrInvalidPattern{Pattern: s, Reason: "empty"}
	}
	if strings.Contains(s, "**") {
		return "", ErrInvalidPattern{Pattern: s, Reason: "** not allowed; use a single * for single-level wildcard"}
	}
	if strings.ContainsAny(s, "?{},[]") {
		return "", ErrInvalidPattern{Pattern: s, Reason: "only `*` wildcards are supported"}
	}
	if strings.Count(s, "*") > 1 {
		return "", ErrInvalidPattern{Pattern: s, Reason: "at most one `*` per pattern"}
	}
	if i := strings.Index(s, "*"); i >= 0 && i != len(s)-1 {
		return "", ErrInvalidPattern{Pattern: s, Reason: "`*` is only supported as a suffix"}
	}
	return PermissionPattern(s), nil
}

// ParsePermissionPath validates and returns a permission path. Refuses empty/whitespace/wildcards.
func ParsePermissionPath(s string) (PermissionPath, error) {
	if s == "" {
		return "", ErrInvalidPattern{Pattern: s, Reason: "empty"}
	}
	if strings.ContainsAny(s, "*?{},[] \t\n") {
		return "", ErrInvalidPattern{Pattern: s, Reason: "permission paths cannot contain wildcards or whitespace"}
	}
	return PermissionPath(s), nil
}
