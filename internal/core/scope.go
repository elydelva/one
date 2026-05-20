package core

// ServiceScope holds the allow/deny permission lists for one service.
type ServiceScope struct {
	Account AccountAlias
	Allow   []PermissionPattern
	Deny    []PermissionPattern
}

// Scope is the parsed representation of .onerc.yaml.
// Default deny: a permission is allowed only when explicitly granted and not denied.
type Scope struct {
	Version  int
	Services map[ServiceID]ServiceScope
}

// Allows reports whether the scope grants the given permission.
// Precedence: deny exact > deny glob > allow exact > allow glob > default deny.
func (s Scope) Allows(perm Permission) bool {
	ss, ok := s.Services[perm.Service]
	if !ok {
		return false
	}

	// Check deny first (deny always wins).
	for _, d := range ss.Deny {
		if d.Matches(perm) {
			return false
		}
	}

	// Then allow.
	for _, a := range ss.Allow {
		if a.Matches(perm) {
			return true
		}
	}

	return false
}

// AccountFor returns the account alias configured for a service, or "default".
func (s Scope) AccountFor(svc ServiceID) AccountAlias {
	if ss, ok := s.Services[svc]; ok && ss.Account != "" {
		return ss.Account
	}
	return "default"
}

// AllowsWithReason returns the same decision as Allows, plus a human-readable reason
// indicating which rule fired (or "default deny" if none).
func (s Scope) AllowsWithReason(perm Permission) (bool, string) {
	ss, ok := s.Services[perm.Service]
	if !ok {
		return false, "default deny (service not in scope)"
	}

	for _, d := range ss.Deny {
		if string(d) == string(perm.Path) {
			return false, "denied (deny exact: " + string(d) + ")"
		}
	}
	for _, d := range ss.Deny {
		if d.Matches(perm) {
			return false, "denied (deny glob: " + string(d) + ")"
		}
	}
	for _, a := range ss.Allow {
		if string(a) == string(perm.Path) {
			return true, "allowed (allow exact: " + string(a) + ")"
		}
	}
	for _, a := range ss.Allow {
		if a.Matches(perm) {
			return true, "allowed (allow glob: " + string(a) + ")"
		}
	}
	return false, "default deny (no matching rule)"
}
