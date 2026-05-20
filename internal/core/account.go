package core

// AccountAlias is the human-readable name for an account (e.g. "default", "work", "perso").
type AccountAlias string

// AccountRef uniquely identifies an account within a service.
type AccountRef struct {
	Service ServiceID
	Alias   AccountAlias
}

// String returns "service:alias".
func (r AccountRef) String() string {
	return string(r.Service) + ":" + string(r.Alias)
}
