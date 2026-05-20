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
